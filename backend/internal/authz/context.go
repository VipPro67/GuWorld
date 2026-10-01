package authz

import "context"

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal AdminPrincipal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (AdminPrincipal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(AdminPrincipal)
	return principal, ok
}
