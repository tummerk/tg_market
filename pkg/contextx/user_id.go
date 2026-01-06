package contextx

import (
	"context"
	"fmt"
)

type UserID string

type contextKeyUserID struct{}

func (u UserID) String() string {
	return string(u)
}

func WithUserID(ctx context.Context, userID UserID) context.Context {
	return context.WithValue(ctx, contextKeyUserID{}, userID)
}

func UserIDFromContext(ctx context.Context) (UserID, error) {
	userID, ok := ctx.Value(contextKeyUserID{}).(UserID)
	if !ok {
		return "", fmt.Errorf("user id: %w", ErrNoValue)
	}

	return userID, nil
}
