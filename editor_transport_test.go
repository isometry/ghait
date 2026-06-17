//go:build !ghait.no_file

package ghait

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// testKeyPEM returns a freshly generated RSA private key in PKCS#1 PEM form,
// suitable for the "file" provider.
func testKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}

// tokenServer stands up an httptest server that answers the installation
// token-mint call, recording the headers and path it observed.
func tokenServer(t *testing.T, seen *http.Header, path *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.Header.Clone()
		*path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "v1.test-installation-token",
			"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestWithRequestEditor_AppliedToMintRequestUnderAuth is the core proof: a
// RequestEditor registered via WithRequestEditor reaches the final
// POST /app/installations/{id}/access_tokens request, and it composes *under*
// ghinstallation's app authentication (the Authorization header is still set).
func TestWithRequestEditor_AppliedToMintRequestUnderAuth(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg, WithURLs(srv.URL, ""), WithRequestEditor(func(req *http.Request) error {
		req.Header.Set("X-Test-Editor", "applied")
		return nil
	}))
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)

	assert.Equal(t, "v1.test-installation-token", tok.GetToken())
	assert.Equal(t, "applied", seen.Get("X-Test-Editor"),
		"request editor header must reach the wire")
	assert.Contains(t, seen.Get("Authorization"), "Bearer ",
		"ghinstallation app auth must still run beneath the editor")
	assert.Contains(t, path, "/app/installations/67890/access_tokens")
}

var errEditorBoom = errors.New("editor boom")

// TestWithRequestEditor_ErrorAbortsRequest verifies that an editor returning
// an error aborts the mint request: NewToken fails with that error and the
// request never reaches the wire.
func TestWithRequestEditor_ErrorAbortsRequest(t *testing.T) {
	var reached atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached.Store(true)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg, WithURLs(srv.URL, ""), WithRequestEditor(func(*http.Request) error {
		return errEditorBoom
	}))
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())

	require.Error(t, err)
	assert.Nil(t, tok)
	require.ErrorIs(t, err, errEditorBoom, "the editor error must propagate to the caller")
	require.ErrorIs(t, err, FatalError{}, "an editor failure is classified non-retryable (fatal)")
	assert.False(t, reached.Load(), "the request must not reach the server when an editor fails")
}

// TestWithRequestEditor_RunInRegistrationOrder verifies multiple editors run
// in the order they were registered.
func TestWithRequestEditor_RunInRegistrationOrder(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	var order []string
	mark := func(id string) RequestEditor {
		return func(*http.Request) error {
			order = append(order, id)
			return nil
		}
	}

	g, err := NewGHAIT(t.Context(), cfg,
		WithURLs(srv.URL, ""),
		WithRequestEditor(mark("first")),
		WithRequestEditor(mark("second")),
	)
	require.NoError(t, err)

	_, err = g.NewToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order)
}

// TestWithRequestEditor_NilEditorIgnored verifies a nil editor passed to
// WithRequestEditor is ignored rather than causing a panic, and real editors
// still apply.
func TestWithRequestEditor_NilEditorIgnored(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg,
		WithURLs(srv.URL, ""),
		WithRequestEditor(nil),
		WithRequestEditor(func(req *http.Request) error {
			req.Header.Set("X-Test-Editor", "applied")
			return nil
		}),
	)
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Equal(t, "applied", seen.Get("X-Test-Editor"))
}

// TestNewGHAIT_NilOptionIgnored verifies a nil Option passed to NewGHAIT is
// ignored rather than causing a panic, and real options still apply.
func TestNewGHAIT_NilOptionIgnored(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg,
		nil,
		WithURLs(srv.URL, ""),
		WithRequestEditor(func(req *http.Request) error {
			req.Header.Set("X-Test-Editor", "applied")
			return nil
		}),
	)
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Equal(t, "applied", seen.Get("X-Test-Editor"))
}

// TestNewGHAIT_NoOptionsPreservesBehaviour is a backward-compatibility guard:
// with no feature options the default no-editor transport path is left
// unchanged and a token is still minted. (WithURLs only sets the base URL, so
// no request editor is registered.)
func TestNewGHAIT_NoOptionsPreservesBehaviour(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg, WithURLs(srv.URL, ""))
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Equal(t, "v1.test-installation-token", tok.GetToken())
	assert.Contains(t, seen.Get("Authorization"), "Bearer ")
}

// TestWithStatelessToken_EnabledSetsHeader verifies WithStatelessToken(true)
// sets X-GitHub-Stateless-S2S-Token: enabled on the mint request and composes
// under ghinstallation auth.
func TestWithStatelessToken_EnabledSetsHeader(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg, WithURLs(srv.URL, ""), WithStatelessToken(true))
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)

	assert.Equal(t, "enabled", seen.Get("X-GitHub-Stateless-S2S-Token"))
	assert.Contains(t, seen.Get("Authorization"), "Bearer ",
		"ghinstallation app auth must still run beneath the option")
	assert.Contains(t, path, "/app/installations/67890/access_tokens")
}

// TestWithStatelessToken_DisabledSetsHeader verifies WithStatelessToken(false)
// sends the legacy-forcing "disabled" value.
func TestWithStatelessToken_DisabledSetsHeader(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg, WithURLs(srv.URL, ""), WithStatelessToken(false))
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)

	assert.Equal(t, "disabled", seen.Get("X-GitHub-Stateless-S2S-Token"))
}

// TestWithStatelessToken_AbsentSendsNoHeader is a backward-compatibility guard:
// without WithStatelessToken the mint request carries no override header and a
// token is still minted (GitHub's automatic rollout decides the format).
func TestWithStatelessToken_AbsentSendsNoHeader(t *testing.T) {
	var seen http.Header
	var path string
	srv := tokenServer(t, &seen, &path)

	cfg := NewConfig(12345, 67890, "file", testKeyPEM(t))

	g, err := NewGHAIT(t.Context(), cfg, WithURLs(srv.URL, ""))
	require.NoError(t, err)

	tok, err := g.NewToken(t.Context())
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Empty(t, seen.Get("X-GitHub-Stateless-S2S-Token"))
}

// roundTripFunc adapts a function to http.RoundTripper for hermetic unit tests
// of editorTransport (no ghinstallation or network required).
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func okResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}
}

// TestEditorTransport_DoesNotMutateOriginalRequest pins the http.RoundTripper
// contract: editors run on a clone, so the request handed to RoundTrip is
// never mutated while the base transport receives the edited clone. It also
// exercises URL mutation, which the RequestEditor godoc advertises as
// supported.
func TestEditorTransport_DoesNotMutateOriginalRequest(t *testing.T) {
	var got *http.Request
	et := &editorTransport{
		base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got = r
			return okResponse(), nil
		}),
		editors: []RequestEditor{
			func(r *http.Request) error {
				r.Header.Set("X-Edited", "yes")
				r.URL.Path = "/rewritten"
				return nil
			},
		},
	}

	orig, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		"https://api.github.com/app/installations/1/access_tokens", nil)
	require.NoError(t, err)
	orig.Header.Set("Authorization", "Bearer original-jwt")

	resp, err := et.RoundTrip(orig)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, orig.Header.Get("X-Edited"),
		"editor must not mutate the original request headers")
	assert.Equal(t, "/app/installations/1/access_tokens", orig.URL.Path,
		"editor must not mutate the original request URL")

	require.NotNil(t, got)
	assert.NotSame(t, orig, got, "base must receive a clone, not the original request")
	assert.Equal(t, "yes", got.Header.Get("X-Edited"))
	assert.Equal(t, "/rewritten", got.URL.Path)
	assert.Equal(t, "Bearer original-jwt", got.Header.Get("Authorization"),
		"the clone must preserve headers set before the editor ran")
}

// TestEditorTransport_LaterEditorErrorAborts verifies a non-first editor's
// error aborts the request before it reaches the base transport.
func TestEditorTransport_LaterEditorErrorAborts(t *testing.T) {
	firstRan := false
	reachedBase := false
	et := &editorTransport{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			reachedBase = true
			return okResponse(), nil
		}),
		editors: []RequestEditor{
			func(*http.Request) error { firstRan = true; return nil },
			func(*http.Request) error { return errEditorBoom },
		},
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.github.com/x", nil)
	require.NoError(t, err)

	resp, err := et.RoundTrip(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	require.Error(t, err)
	assert.Nil(t, resp)
	require.ErrorIs(t, err, errEditorBoom)
	assert.True(t, firstRan, "the first editor should run before the second errors")
	assert.False(t, reachedBase, "an aborted request must not reach the base transport")
}

// TestEditorTransport_ConcurrentReuseIsRaceFree pins that the transport, whose
// editors slice is read-only after construction, is safe under concurrent
// reuse across goroutines. Meaningful under `go test -race`.
func TestEditorTransport_ConcurrentReuseIsRaceFree(t *testing.T) {
	var count atomic.Int64
	et := &editorTransport{
		base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			count.Add(1)
			if r.Header.Get("X-Editor") != "on" {
				return nil, errors.New("editor header missing on the wire")
			}
			return okResponse(), nil
		}),
		editors: []RequestEditor{
			func(r *http.Request) error { r.Header.Set("X-Editor", "on"); return nil },
		},
	}

	ctx := t.Context()
	const n = 32
	var g errgroup.Group
	for range n {
		g.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/x", nil)
			if err != nil {
				return err
			}
			resp, err := et.RoundTrip(req)
			if err != nil {
				return err
			}
			return resp.Body.Close()
		})
	}
	require.NoError(t, g.Wait())
	assert.Equal(t, int64(n), count.Load())
}
