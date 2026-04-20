package admission

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
	"grid_trade/internal/limit"
)

func TestAdmissionChecksRateLimitBeforeRisk(t *testing.T) {
	svc := NewService(fakeRules{}, fakeLimiter{err: limit.ErrRateLimited}, fakeRisk{})

	err := svc.Admit(domain.OrderIntent{ClientOrderID: "cid-1"})
	require.ErrorIs(t, err, limit.ErrRateLimited)
}

type fakeRules struct{}

func (fakeRules) Validate(domain.OrderIntent) error { return nil }

type fakeLimiter struct{ err error }

func (f fakeLimiter) Allow(string) error { return f.err }

type fakeRisk struct{}

func (fakeRisk) Check(domain.OrderIntent) error {
	return errors.New("risk should not be reached")
}
