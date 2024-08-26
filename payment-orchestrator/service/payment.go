package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/illenko/observability-common/tracing"
	"github.com/illenko/payment-orchestrator/model"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/propagation"
)

type PaymentService struct {
	client      http.Client
	restyClient *resty.Client
}

func NewPaymentService(client http.Client, restyClient *resty.Client) *PaymentService {
	return &PaymentService{client: client, restyClient: restyClient}
}

func (s *PaymentService) CallRoutingService(ctx context.Context, routeID string) (model.RoutingResponse, error) {
	routingCtx, routingSpan := tracing.T.Start(ctx, "routingServiceCall")
	defer routingSpan.End()

	routingURL := fmt.Sprintf("http://localhost:8081/routings/%s", routeID)
	req, err := http.NewRequestWithContext(routingCtx, "GET", routingURL, nil)
	if err != nil {
		return model.RoutingResponse{}, fmt.Errorf("failed to create routing service request: %w", err)
	}

	propagation.TraceContext{}.Inject(routingCtx, propagation.HeaderCarrier(req.Header))

	log.Println("Routing service call started")
	resp, err := s.client.Do(req)
	if err != nil {
		return model.RoutingResponse{}, fmt.Errorf("failed to call routing service: %w", err)
	}
	defer resp.Body.Close()
	log.Println("Routing service call completed")

	if resp.StatusCode != http.StatusOK {
		return model.RoutingResponse{}, fmt.Errorf("routing service returned an error: %s", resp.Status)
	}

	var routingResp model.RoutingResponse
	if err := json.NewDecoder(resp.Body).Decode(&routingResp); err != nil {
		return model.RoutingResponse{}, fmt.Errorf("failed to decode routing service response: %w", err)
	}
	log.Printf("Routing service response: %+v\n", routingResp)

	return routingResp, nil
}

func (s *PaymentService) CallPaymentProviderXService(ctx context.Context, url string, payRequest model.PaymentProviderRequest) (model.PaymentProviderResponse, error) {
	paymentProviderCtx, paymentProviderSpan := tracing.T.Start(ctx, "paymentProviderServiceCall")
	defer paymentProviderSpan.End()

	s.restyClient.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		propagation.TraceContext{}.Inject(paymentProviderCtx, propagation.HeaderCarrier(req.Header))
		return nil
	})

	log.Println("Payment provider service call started")
	resp, err := s.restyClient.R().
		SetContext(paymentProviderCtx).
		SetHeader("Content-Type", "application/json").
		SetBody(payRequest).
		Post(url)
	if err != nil {
		return model.PaymentProviderResponse{}, fmt.Errorf("failed to call payment provider service: %w", err)
	}
	log.Println("Payment provider service call completed")

	if resp.StatusCode() != http.StatusOK {
		return model.PaymentProviderResponse{}, fmt.Errorf("payment provider service returned an error: %s", resp.Status())
	}

	var paymentProviderResp model.PaymentProviderResponse
	if err := json.Unmarshal(resp.Body(), &paymentProviderResp); err != nil {
		return model.PaymentProviderResponse{}, fmt.Errorf("failed to decode payment provider response: %w", err)
	}

	return paymentProviderResp, nil
}
