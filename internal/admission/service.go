package admission

import (
	"errors"

	"grid_trade/internal/domain"
)

type RuleValidator interface {
	Validate(domain.OrderIntent) error
}

type Limiter interface {
	Allow(string) error
}

type RiskChecker interface {
	Check(domain.OrderIntent) error
}

type Service struct {
	rules   RuleValidator
	limiter Limiter
	risk    RiskChecker
}

func NewService(rules RuleValidator, limiter Limiter, risk RiskChecker) Service {
	return Service{
		rules:   rules,
		limiter: limiter,
		risk:    risk,
	}
}

func (s Service) Admit(intent domain.OrderIntent) error {
	if intent.ClientOrderID == "" {
		return errors.New("clientOrderID is required")
	}
	if err := s.rules.Validate(intent); err != nil {
		return err
	}
	if err := s.limiter.Allow("orders"); err != nil {
		return err
	}
	if err := s.risk.Check(intent); err != nil {
		return err
	}
	return nil
}
