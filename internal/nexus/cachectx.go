package nexus

import "context"

type cacheBypassCtxKey struct{}

// WithCacheBypass returns ctx that makes withCache skip reading the cache but still
// refresh the entry after a successful fetch.
func WithCacheBypass(ctx context.Context, bypass bool) context.Context {
	if !bypass {
		return ctx
	}
	return context.WithValue(ctx, cacheBypassCtxKey{}, true)
}

func cacheBypassFrom(ctx context.Context) bool {
	v, ok := ctx.Value(cacheBypassCtxKey{}).(bool)
	return ok && v
}
