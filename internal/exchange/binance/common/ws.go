package common

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"

	"grid_trade/internal/domain"
)

type WSDialer interface {
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

type UserStreamConnector struct {
	endpoints Endpoints
	dialer    WSDialer
}

func NewUserStreamConnector(endpoints Endpoints, dialer WSDialer) *UserStreamConnector {
	if dialer == nil {
		dialer = &websocket.Dialer{}
	}
	return &UserStreamConnector{
		endpoints: endpoints,
		dialer:    dialer,
	}
}

func (c *UserStreamConnector) Connect(ctx context.Context, market domain.MarketType, listenKey string) (io.Closer, error) {
	url, err := c.endpoints.UserStreamURL(market, listenKey)
	if err != nil {
		return nil, err
	}

	conn, resp, err := c.dialer.DialContext(ctx, url, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("connect user stream failed with status %d: %w", resp.StatusCode, err)
		}
		return nil, err
	}
	return conn, nil
}
