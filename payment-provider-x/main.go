package main

import (
	"context"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/Cyprinus12138/otelgin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/illenko/observability-common"
	"github.com/joho/godotenv"
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

	if err := godotenv.Load(); err != nil {
		slog.ErrorContext(ctx, "Error loading .env file", slog.Any("error", err))
		return
	}

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
	router.Use(otelgin.Middleware("payment-provider-x"))

	router.POST("/pay", paymentHandler)
	slog.ErrorContext(ctx, "server failed", slog.Any("error", router.Run(":"+os.Getenv("PORT"))))
}

func paymentHandler(c *gin.Context) {
	var paymentReq PaymentRequest

	if err := c.ShouldBindJSON(&paymentReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	slog.InfoContext(c.Request.Context(), "Payment processing started")
	randomDelay := time.Duration(200+rand.Intn(2800)) * time.Millisecond
	time.Sleep(randomDelay)
	slog.InfoContext(c.Request.Context(), "Payment processing completed")

	statuses := []string{"success", "failed"}

	randomStatus := statuses[rand.Intn(len(statuses))]

	response := PaymentResponse{
		PaymentID: uuid.New().String(),
		Status:    randomStatus,
	}

	c.JSON(http.StatusOK, response)
}
