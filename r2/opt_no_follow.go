package r2

import "net/http"

// OptNoFollow tells the http client to not follow redirects.
func OptNoFollow() Option {
	return func(r *Request) error {
		r.Client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	}
}