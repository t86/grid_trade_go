package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestStoreUpsertOrder(t *testing.T) {
	store := NewStore()
	order := domain.OrderRecord{
		ClientOrderID: "cid-1",
		Symbol:        "BTCUSDT",
		Status:        "NEW",
	}

	store.UpsertOrder(order)

	got, ok := store.Order("cid-1")
	require.True(t, ok)
	require.Equal(t, order, got)
}
