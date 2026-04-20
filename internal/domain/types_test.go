package domain

import "testing"

func TestMarketTypeValues(t *testing.T) {
	if MarketSpot != "spot" {
		t.Fatalf("expected MarketSpot to be spot, got %q", MarketSpot)
	}
	if MarketFuturesUM != "futures_um" {
		t.Fatalf("expected MarketFuturesUM to be futures_um, got %q", MarketFuturesUM)
	}
}
