package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// RedirectToHTTPS redirects any HTTP requests to HTTPS depending on the issuer's scheme
func RedirectToHTTPS(issuer string) func(next http.Handler) http.Handler {
	if strings.HasPrefix(issuer, "https://") {
		u, _ := url.Parse(issuer)

		return func(next http.Handler) http.Handler {
			return &redirectToHttps{u, next}
		}
	} else {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
}

type redirectToHttps struct {
	issuer *url.URL
	next   http.Handler
}

var _ http.Handler = (*redirectToHttps)(nil)

func (h *redirectToHttps) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil {
		h.next.ServeHTTP(w, r)
		return
	}

	dest := h.issuer.JoinPath(r.URL.Path)
	dest.RawQuery = r.URL.RawQuery

	w.Header().Set("Location", dest.String())
	w.WriteHeader(http.StatusTemporaryRedirect)
}
