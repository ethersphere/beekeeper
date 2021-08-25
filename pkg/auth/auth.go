package auth

import "context"

type tokenKey string

const authTokenKey = tokenKey("auth")

func GetAuthToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(authTokenKey).(string)
	return token, ok
}

func WithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey, token)
}
