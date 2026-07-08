package admin

import (
	"net/http"
	"net/url"
	"time"

	"github.com/esrid/maison/internal/auth"
)

const (
	cookieName = "admin_session"
	sessionTTL = 12 * time.Hour
	loginPath  = "/admin/login"
)

// RequireAuth resolves the admin session cookie and attaches the logged-in
// User to the request context, or redirects to the login page (preserving
// the original destination) if there isn't a valid one.
func RequireAuth(store *Store, signer auth.Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
			if err != nil {
				redirectToLogin(w, r)
				return
			}
			token, err := signer.Verify(c.Value)
			if err != nil {
				redirectToLogin(w, r)
				return
			}
			user, err := store.FindSession(r.Context(), auth.HashToken(token))
			if err != nil {
				redirectToLogin(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
		})
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	dest := loginPath
	if r.URL.Path != "" && r.URL.Path != "/admin" {
		dest += "?next=" + url.QueryEscape(r.URL.Path)
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}
