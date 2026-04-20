package limit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimiterRejectsWhenTokensExhausted(t *testing.T) {
	limiter := NewTokenBucket(1, time.Minute)

	require.NoError(t, limiter.Allow("orders"))
	require.ErrorIs(t, limiter.Allow("orders"), ErrRateLimited)
}
