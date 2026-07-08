package admin

import "context"

type ctxKey struct{}

func withUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// UserFromContext returns the logged-in admin for the current request.
// Panics if called outside RequireAuth — every protected admin route is
// wrapped by it, so a missing value means a route was mounted wrong.
func UserFromContext(ctx context.Context) User {
	u, ok := ctx.Value(ctxKey{}).(User)
	if !ok {
		panic("admin: UserFromContext called on a request with no admin auth middleware")
	}
	return u
}
