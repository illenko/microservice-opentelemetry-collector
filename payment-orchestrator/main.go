package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/propagation"
	"io"
	"log"
	"net/http"

	"github.com/illenko/observability-common/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var client http.Client
var restyClient *resty.Client

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

	client = http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	restyClient = resty.New()
	restyClient.SetDebug(true)

	mux := http.NewServeMux()
	mux.Handle("POST /payments", otelhttp.NewHandler(http.HandlerFunc(paymentHandler), "payment", otelhttp.WithPropagators(propagation.TraceContext{})))
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	paymentReq, err := readPaymentRequest(r)
	if err != nil {
		writeErrorResponse(w, "Invalid request", err)
		return
	}
	log.Printf("Received payment request: %+v\n", paymentReq)

	routingResp, err := callRoutingService(ctx, paymentReq.RouteID)
	if err != nil {
		writeErrorResponse(w, "Failed to call routing service", err)
		return
	}
	log.Printf("Routing service response: %+v\n", routingResp)

	paymentProviderReq := PaymentProviderRequest{
		OrderID:  paymentReq.OrderID,
		Amount:   paymentReq.Amount,
		Currency: paymentReq.Currency,
	}

	paymentProviderResp, err := callPaymentProviderXService(ctx, routingResp.URL, paymentProviderReq)
	if err != nil {
		writeErrorResponse(w, "Failed to call payment provider service", err)
		return
	}

	response := PaymentResponse{
		OrderID:   paymentReq.OrderID,
		PaymentID: paymentProviderResp.PaymentID,
		Status:    paymentProviderResp.Status,
	}

	writeSuccessResponse(w, response)
}

func writeErrorResponse(w http.ResponseWriter, message string, err error) {
	log.Printf("%s: %v", message, err)
	response := map[string]string{"status": "error", "message": message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func writeSuccessResponse(w http.ResponseWriter, res PaymentResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
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

func callPaymentProviderXService(ctx context.Context, url string, payRequest PaymentProviderRequest) (PaymentProviderResponse, error) {
	paymentProviderCtx, paymentProviderSpan := tracing.T.Start(ctx, "paymentProviderXServiceCall")
	defer paymentProviderSpan.End()

	restyClient.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		propagation.TraceContext{}.Inject(paymentProviderCtx, propagation.HeaderCarrier(req.Header))
		return nil
	})

	log.Println("Payment provider X service call started")
	resp, err := restyClient.R().
		SetContext(paymentProviderCtx).
		SetHeader("Content-Type", "application/json").
		SetBody(payRequest).
		Post(url)
	if err != nil {
		return PaymentProviderResponse{}, fmt.Errorf("failed to call payment provider service: %w", err)
	}
	log.Println("Payment provider X service call completed")

	if resp.StatusCode() != http.StatusOK {
		return PaymentProviderResponse{}, fmt.Errorf("payment provider service returned an error: %s", resp.Status())
	}

	var paymentProviderResp PaymentProviderResponse
	if err := json.Unmarshal(resp.Body(), &paymentProviderResp); err != nil {
		return PaymentProviderResponse{}, fmt.Errorf("failed to decode payment provider response: %w", err)
	}

	return paymentProviderResp, nil
}
