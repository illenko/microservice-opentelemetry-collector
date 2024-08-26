package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/illenko/payment-orchestrator/model"
	"github.com/illenko/payment-orchestrator/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) Payment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paymentReq, err := ReadPaymentRequest(r)
	if err != nil {
		WriteErrorResponse(w, "Invalid request", err)
		return
	}
	log.Printf("Received payment request: %+v\n", paymentReq)

	routingResp, err := h.service.CallRoutingService(ctx, paymentReq.RouteID)
	if err != nil {
		WriteErrorResponse(w, "Failed to call routing service", err)
		return
	}
	log.Printf("Routing service response: %+v\n", routingResp)

	paymentProviderReq := model.PaymentProviderRequest{
		OrderID:  paymentReq.OrderID,
		Amount:   paymentReq.Amount,
		Currency: paymentReq.Currency,
	}

	paymentProviderResp, err := h.service.CallPaymentProviderXService(ctx, routingResp.URL, paymentProviderReq)
	if err != nil {
		WriteErrorResponse(w, "Failed to call payment provider service", err)
		return
	}

	response := model.PaymentResponse{
		OrderID:   paymentReq.OrderID,
		PaymentID: paymentProviderResp.PaymentID,
		Status:    paymentProviderResp.Status,
	}

	WriteSuccessResponse(w, response)
}

func WriteErrorResponse(w http.ResponseWriter, message string, err error) {
	log.Printf("%s: %v", message, err)
	response := map[string]string{"status": "error", "message": message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func WriteSuccessResponse(w http.ResponseWriter, res model.PaymentResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func ReadPaymentRequest(r *http.Request) (model.PaymentRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return model.PaymentRequest{}, fmt.Errorf("unable to read request body: %w", err)
	}
	defer r.Body.Close()

	var paymentReq model.PaymentRequest
	if err := json.Unmarshal(body, &paymentReq); err != nil {
		return model.PaymentRequest{}, fmt.Errorf("invalid request body: %w", err)
	}
	return paymentReq, nil
}
