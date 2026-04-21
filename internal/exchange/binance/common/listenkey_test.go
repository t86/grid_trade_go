package common

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestCreateListenKeyUsesExpectedEndpointAndHeader(t *testing.T) {
	t.Run("spot", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/api/v3/userDataStream", r.URL.Path)
			require.Equal(t, "api-key", r.Header.Get("X-MBX-APIKEY"))
			_, _ = io.WriteString(w, `{"listenKey":"spot-key"}`)
		}))
		defer server.Close()

		client := NewRESTClient("api-key", server.Client(), Endpoints{
			SpotRESTBaseURL:    server.URL,
			FuturesRESTBaseURL: server.URL,
		})

		key, err := client.CreateListenKey(context.Background(), domain.MarketSpot)
		require.NoError(t, err)
		require.Equal(t, "spot-key", key)
	})

	t.Run("futures", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/fapi/v1/listenKey", r.URL.Path)
			require.Equal(t, "api-key", r.Header.Get("X-MBX-APIKEY"))
			_, _ = io.WriteString(w, `{"listenKey":"futures-key"}`)
		}))
		defer server.Close()

		client := NewRESTClient("api-key", server.Client(), Endpoints{
			SpotRESTBaseURL:    server.URL,
			FuturesRESTBaseURL: server.URL,
		})

		key, err := client.CreateListenKey(context.Background(), domain.MarketFuturesUM)
		require.NoError(t, err)
		require.Equal(t, "futures-key", key)
	})
}
