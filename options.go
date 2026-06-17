package ghait

import "net/http"

// RequestEditor mutates an outbound HTTP request before it is sent by the
// ghait client. It is invoked for every request the client makes, including
// the POST /app/installations/{id}/access_tokens token-mint request, after the
// GitHub App JWT Authorization header has been attached and immediately before
// the request goes on the wire. The request passed in is a clone: an editor
// may mutate headers, URL, and trailers but must not read or close req.Body.
// Editors run in registration order; the first non-nil error aborts the
// request and is returned to the caller.
type RequestEditor func(*http.Request) error

// Option configures optional NewGHAIT behaviour. The empty set of options
// preserves the historical behaviour exactly.
type Option func(*options)

type options struct {
	requestEditors []RequestEditor
	baseURL        *string
	uploadURL      *string
}

// WithURLs sets the base and upload URLs of the underlying GitHub API client,
// for example to target a GitHub Enterprise Server instance or a test server.
// An empty string leaves the corresponding URL at its default. Each URL is
// used as given (only its format is validated); no "/api/v3/" or
// "/api/uploads/" suffix is appended. When this option is not supplied, the
// public GitHub API endpoints are used.
func WithURLs(baseURL, uploadURL string) Option {
	return func(o *options) {
		if baseURL != "" {
			o.baseURL = &baseURL
		}
		if uploadURL != "" {
			o.uploadURL = &uploadURL
		}
	}
}

// WithRequestEditor registers a RequestEditor. It may be supplied multiple
// times; editors run in the order registered. A nil editor is ignored.
func WithRequestEditor(fn RequestEditor) Option {
	return func(o *options) {
		if fn != nil {
			o.requestEditors = append(o.requestEditors, fn)
		}
	}
}

// statelessTokenHeader is GitHub's per-request override that selects the
// installation access token format. See:
// https://github.blog/changelog/2026-05-15-github-app-installation-tokens-per-request-override-header/
const statelessTokenHeader = "X-GitHub-Stateless-S2S-Token"

// WithStatelessToken sets the "X-GitHub-Stateless-S2S-Token" header on the
// installation token-mint request issued by NewToken, NewInstallationToken,
// and NewTokenWithOptions:
//
//   - WithStatelessToken(true)  sends "enabled"  (stateless JWT token)
//   - WithStatelessToken(false) sends "disabled" (legacy opaque token)
//
// When this option is not supplied, no header is sent and behaviour is
// unchanged.
func WithStatelessToken(enabled bool) Option {
	value := "disabled"
	if enabled {
		value = "enabled"
	}
	return WithRequestEditor(func(r *http.Request) error {
		r.Header.Set(statelessTokenHeader, value)
		return nil
	})
}
