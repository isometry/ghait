package ghait

import "net/http"

// editorTransport runs caller-supplied RequestEditors on a cloned request,
// honouring the http.RoundTripper contract (it must not mutate the request it
// is handed). It is installed as the base RoundTripper beneath ghinstallation's
// AppsTransport and the rate-limit waiter, so editors observe the final,
// JWT-signed request and never bypass authentication or rate limiting.
type editorTransport struct {
	base    http.RoundTripper
	editors []RequestEditor
}

func (t *editorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	for _, edit := range t.editors {
		if err := edit(r); err != nil {
			return nil, err
		}
	}
	return t.base.RoundTrip(r)
}
