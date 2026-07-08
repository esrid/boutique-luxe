package cart

import "context"

type ctxKey struct{}

func withCartID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IDFromContext returns the current request's cart ID. Every storefront
// route is wrapped by Middleware, which always sets this — a missing value
// means a route was mounted outside that middleware, a wiring bug worth
// failing loudly on rather than silently operating on cart ID 0.
func IDFromContext(ctx context.Context) int64 {
	id, ok := ctx.Value(ctxKey{}).(int64)
	if !ok {
		panic("cart: IDFromContext called on a request with no cart middleware")
	}
	return id
}
