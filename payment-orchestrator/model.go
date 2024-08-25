package main

type PaymentRequest struct {
	OrderID  string  `json:"orderId"`
	RouteID  string  `json:"routeId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RoutingResponse struct {
	PaymentProvider string `json:"paymentProvider"`
	URL             string `json:"url"`
}
