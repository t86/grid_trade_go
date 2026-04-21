package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"grid_trade/internal/domain"
)

type WSRequest struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

func BuildOrderPlaceRequest(id string, intent domain.OrderIntent, apiKey, secret string, now time.Time) (WSRequest, error) {
	params := map[string]any{
		"apiKey":    apiKey,
		"symbol":    intent.Symbol,
		"side":      intent.Side,
		"type":      intent.OrderType,
		"quantity":  intent.Quantity,
		"timestamp": now.UnixMilli(),
	}
	if intent.Price != "" {
		params["price"] = intent.Price
	}
	if intent.TimeInForce != "" {
		params["timeInForce"] = intent.TimeInForce
	}
	if intent.PositionSide != "" {
		params["positionSide"] = intent.PositionSide
	}
	if intent.ReduceOnly {
		params["reduceOnly"] = "true"
	}
	if intent.ClientOrderID != "" {
		params["newClientOrderId"] = intent.ClientOrderID
	}

	signature, err := Sign(secret, params)
	if err != nil {
		return WSRequest{}, err
	}
	params["signature"] = signature

	return WSRequest{
		ID:     id,
		Method: "order.place",
		Params: params,
	}, nil
}

func Sign(secret string, params map[string]any) (string, error) {
	query, err := canonicalQuery(params)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(query)); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func canonicalQuery(params map[string]any) (string, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := url.Values{}
	for _, key := range keys {
		value, err := stringifyParam(params[key])
		if err != nil {
			return "", fmt.Errorf("stringify %s: %w", key, err)
		}
		values.Set(key, value)
	}
	return values.Encode(), nil
}

func stringifyParam(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return fmt.Sprint(v), nil
	}
}
