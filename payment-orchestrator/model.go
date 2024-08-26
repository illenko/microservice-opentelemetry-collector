package main

type PaymentRequest struct {
	OrderID  string  `json:"orderId"`
	RouteID  string  `json:"routeId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentResponse struct {
	PaymentID string `json:"paymentId"`
	OrderID   string `json:"orderId"`
	Status    string `json:"status"`
}

type RoutingResponse struct {
	PaymentProvider string `json:"paymentProvider"`
	URL             string `json:"url"`
}

type PaymentProviderRequest struct {
	OrderID  string  `json:"orderId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentProviderResponse struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
}
