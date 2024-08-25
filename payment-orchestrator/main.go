package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/propagation"

	"github.com/illenko/observability-common/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var client = http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

func main() {
	ctx := context.Background()

	tp, err := tracing.InitProvider("localhost:4317", "payment-orchestrator", ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("POST /payments", otelhttp.NewHandler(http.HandlerFunc(paymentHandler), "payment", otelhttp.WithPropagators(propagation.TraceContext{})))
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paymentReq, err := readPaymentRequest(r)
	if err != nil {
		writeJSONResponse(w, "error", fmt.Sprintf("Invalid request: %v", err))
		return
	}
	log.Printf("Received payment request: %+v\n", paymentReq)

	routingResp, err := callRoutingService(ctx, paymentReq.RouteID)
	if err != nil {
		writeJSONResponse(w, "error", fmt.Sprintf("Failed to call routing service: %v", err))
		return
	}
	log.Printf("Routing service response: %+v\n", routingResp)

	if err := callPaymentProviderXService(ctx); err != nil {
		writeJSONResponse(w, "error", fmt.Sprintf("Failed to call payment provider service: %v", err))
		return
	}

	writeJSONResponse(w, "success", "Payment processed")
}

func writeJSONResponse(w http.ResponseWriter, status string, message string) {
	response := PaymentResponse{
		Status:  status,
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func readPaymentRequest(r *http.Request) (PaymentRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return PaymentRequest{}, fmt.Errorf("unable to read request body: %w", err)
	}
	defer r.Body.Close()

	var paymentReq PaymentRequest
	if err := json.Unmarshal(body, &paymentReq); err != nil {
		return PaymentRequest{}, fmt.Errorf("invalid request body: %w", err)
	}
	return paymentReq, nil
}

func callRoutingService(ctx context.Context, routeID string) (RoutingResponse, error) {
	routingCtx, routingSpan := tracing.T.Start(ctx, "routingServiceCall")
	defer routingSpan.End()

	routingURL := fmt.Sprintf("http://localhost:8081/routings/%s", routeID)
	req, err := http.NewRequestWithContext(routingCtx, "GET", routingURL, nil)
	if err != nil {
		return RoutingResponse{}, fmt.Errorf("failed to create routing service request: %w", err)
	}

	propagation.TraceContext{}.Inject(routingCtx, propagation.HeaderCarrier(req.Header))

	log.Println("Routing service call started")
	resp, err := client.Do(req)
	if err != nil {
		return RoutingResponse{}, fmt.Errorf("failed to call routing service: %w", err)
	}
	defer resp.Body.Close()
	log.Println("Routing service call completed")

	if resp.StatusCode != http.StatusOK {
		return RoutingResponse{}, fmt.Errorf("routing service returned an error: %s", resp.Status)
	}

	var routingResp RoutingResponse
	if err := json.NewDecoder(resp.Body).Decode(&routingResp); err != nil {
		return RoutingResponse{}, fmt.Errorf("failed to decode routing service response: %w", err)
	}
	log.Printf("Routing service response: %+v\n", routingResp)

	return routingResp, nil
}

func callPaymentProviderXService(ctx context.Context) error {
	_, paymentProviderSpan := tracing.T.Start(ctx, "paymentProviderXServiceCall")
	defer paymentProviderSpan.End()

	log.Println("Payment provider X service call started")
	time.Sleep(200 * time.Millisecond)
	log.Println("Payment provider X service call completed")

	return nil
}
