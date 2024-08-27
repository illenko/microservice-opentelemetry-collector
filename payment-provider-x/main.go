package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/illenko/observability-common/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/propagation"
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

	tp, _, err := tracing.InitProvider("localhost:4317", "payment-provider-x", ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	router := gin.Default()
	router.Use(otelgin.Middleware("payment-provider-x", otelgin.WithTracerProvider(tp), otelgin.WithPropagators(propagation.TraceContext{})))
	router.POST("/pay", paymentHandler)
	log.Fatal(router.Run(":8082"))
}

func paymentHandler(c *gin.Context) {
	ctx := c.Request.Context()
	var paymentReq PaymentRequest

	if err := c.ShouldBindJSON(&paymentReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx, paymentSpan := tracing.T.Start(ctx, "processPayment")
	defer paymentSpan.End()

	log.Println("Payment processing started")
	time.Sleep(200 * time.Millisecond) // Simulate processing delay
	log.Println("Payment processing completed")

	response := PaymentResponse{
		PaymentID: "12345",
		Status:    "success",
	}

	c.JSON(http.StatusOK, response)
}
