package mcp

import "context"

type authContextKey string

const requestAuthKey authContextKey = "mcpRequestAuth"

type RequestAuth struct {
	UserID             string
	Challenge          string
	ChallengeMessage   string
	ChallengeErrorCode string
}

func WithRequestAuth(ctx context.Context, auth RequestAuth) context.Context {
	return context.WithValue(ctx, requestAuthKey, auth)
}

func RequestAuthFromContext(ctx context.Context) RequestAuth {
	auth, _ := ctx.Value(requestAuthKey).(RequestAuth)
	return auth
}
