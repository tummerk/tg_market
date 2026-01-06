package contextx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go-backend-example/pkg/contextx"
)

func TestUserID(t *testing.T) {
	rq := require.New(t)
	ctx := context.Background()

	var testUserIDEmpty contextx.UserID

	testUserIDNotEmpty := contextx.UserID("test-user-id")

	userID, err := contextx.UserIDFromContext(ctx)
	rq.Equal(testUserIDEmpty, userID)
	rq.ErrorIs(err, contextx.ErrNoValue)
	rq.ErrorContains(err, "user id: no value in context")

	ctx = contextx.WithUserID(ctx, testUserIDNotEmpty)

	userID, err = contextx.UserIDFromContext(ctx)
	rq.Equal(testUserIDNotEmpty, userID)
	rq.NoError(err)
}
