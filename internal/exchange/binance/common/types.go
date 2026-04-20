package common

type WSOrderAck struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Result struct {
		Symbol        string `json:"symbol"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"result"`
}
