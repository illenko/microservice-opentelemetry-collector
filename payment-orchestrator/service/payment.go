package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/illenko/payment-orchestrator/model"

	"github.com/go-resty/resty/v2"
)

type PaymentService struct {
	client      http.Client
	restyClient *resty.Client
}

func NewPaymentService(client http.Client, restyClient *resty.Client) *PaymentService {
	return &PaymentService{client: client, restyClient: restyClient}
}

func (s *PaymentService) CallRoutingService(ctx context.Context, routeID string) (model.RoutingResponse, error) {

	routingURL := fmt.Sprintf("http://localhost:8081/routings/%s", routeID)
	req, err := http.NewRequestWithContext(ctx, "GET", routingURL, nil)
	if err != nil {
		return model.RoutingResponse{}, fmt.Errorf("failed to create routing service request: %w", err)
	}

	slog.InfoContext(ctx, "Routing service call started")
	resp, err := s.client.Do(req)
	if err != nil {
		return model.RoutingResponse{}, fmt.Errorf("failed to call routing service: %w", err)
	}
	defer resp.Body.Close()
	slog.InfoContext(ctx, "Routing service call completed")

	if resp.StatusCode != http.StatusOK {
		return model.RoutingResponse{}, fmt.Errorf("routing service returned an error: %s", resp.Status)
	}

	var routingResp model.RoutingResponse
	if err := json.NewDecoder(resp.Body).Decode(&routingResp); err != nil {
		return model.RoutingResponse{}, fmt.Errorf("failed to decode routing service response: %w", err)
	}
	slog.InfoContext(ctx, "Routing service response", slog.Any("response", routingResp))

	return routingResp, nil
}

func (s *PaymentService) CallPaymentProviderXService(ctx context.Context, url string, payRequest model.PaymentProviderRequest) (model.PaymentProviderResponse, error) {

	slog.InfoContext(ctx, "Payment provider service call started")
	resp, err := s.restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payRequest).
		Post(url)
	if err != nil {
		return model.PaymentProviderResponse{}, fmt.Errorf("failed to call payment provider service: %w", err)
	}
	slog.InfoContext(ctx, "Payment provider service call completed")

	if resp.StatusCode() != http.StatusOK {
		return model.PaymentProviderResponse{}, fmt.Errorf("payment provider service returned an error: %s", resp.Status())
	}

	var paymentProviderResp model.PaymentProviderResponse
	if err := json.Unmarshal(resp.Body(), &paymentProviderResp); err != nil {
		return model.PaymentProviderResponse{}, fmt.Errorf("failed to decode payment provider response: %w", err)
	}

	return paymentProviderResp, nil
}
