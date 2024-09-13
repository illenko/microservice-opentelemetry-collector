package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/illenko/observability-common"
	"go.opentelemetry.io/otel"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type PaymentRequest struct {
	OrderID  string  `json:"orderId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentResponse struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
}

func main() {
	ctx := context.Background()

	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	os.Setenv("OTEL_SERVICE_NAME", "payment-provider-x")

	observability.SetupLogging()

	shutdown, err := observability.SetupOpenTelemetry(ctx)

	if err != nil {
		slog.ErrorContext(ctx, "error setting up OpenTelemetry", slog.Any("error", err))
	}

	if shutdown != nil {
		defer func() {
			if err := shutdown(ctx); err != nil {
				slog.ErrorContext(ctx, "error during shutdown", slog.Any("error", err))
			}
		}()
	}

	router := gin.Default()
	router.Use(otelgin.Middleware("payment-provider-x",
		otelgin.WithTracerProvider(otel.GetTracerProvider()),
		otelgin.WithPropagators(otel.GetTextMapPropagator())))
	router.POST("/pay", paymentHandler)
	slog.ErrorContext(ctx, "server failed", slog.Any("error", router.Run(":8082")))
}

func paymentHandler(c *gin.Context) {
	var paymentReq PaymentRequest

	if err := c.ShouldBindJSON(&paymentReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	slog.InfoContext(c.Request.Context(), "Payment processing started")
	time.Sleep(200 * time.Millisecond) // Simulate processing delay
	slog.InfoContext(c.Request.Context(), "Payment processing completed")

	response := PaymentResponse{
		PaymentID: "12345",
		Status:    "success",
	}

	c.JSON(http.StatusOK, response)
}
