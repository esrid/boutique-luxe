package cart

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/esrid/maison/internal/auth"
)

const cookieName = "cart_token"
const cookieTTL = 30 * 24 * time.Hour

// Middleware resolves the caller's cart from a signed cookie, creating one
// (and setting the cookie) if it's missing, invalid, or points at a cart
// that no longer exists. The resolved cart ID is attached to the request
// context — see IDFromContext.
func Middleware(store *Store, signer auth.Signer, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cartID, token, isNew, err := resolve(r, store, signer)
			if err != nil {
				slog.Error("resolve cart", "err", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if isNew {
				http.SetCookie(w, &http.Cookie{
					Name:     cookieName,
					Value:    signer.Sign(token),
					Path:     "/",
					HttpOnly: true,
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
					Expires:  time.Now().Add(cookieTTL),
				})
			}
			next.ServeHTTP(w, r.WithContext(withCartID(r.Context(), cartID)))
		})
	}
}

func resolve(r *http.Request, store *Store, signer auth.Signer) (cartID int64, token string, isNew bool, err error) {
	if c, cerr := r.Cookie(cookieName); cerr == nil {
		if tok, verr := signer.Verify(c.Value); verr == nil {
			if id, ferr := store.FindByTokenHash(r.Context(), auth.HashToken(tok)); ferr == nil {
				return id, tok, false, nil
			}
		}
	}

	tok, err := auth.GenerateToken()
	if err != nil {
		return 0, "", false, err
	}
	id, err := store.Create(r.Context(), auth.HashToken(tok))
	if err != nil {
		return 0, "", false, err
	}
	return id, tok, true, nil
}
