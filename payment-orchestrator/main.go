package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/illenko/observability-common/tracing"
)

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
	mux.HandleFunc("/payment", tracing.HandlerMiddleware(paymentHandler))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.T.Start(r.Context(), "paymentHandler")
	defer span.End()

	// Simulate routing-service call
	_, routingSpan := tracing.T.Start(ctx, "routingServiceCall")
	// Simulate routing-service processing
	log.Println("Routing service call started")
	time.Sleep(100 * time.Millisecond) // Simulate delay
	log.Println("Routing service call completed")
	routingSpan.End()

	// Simulate payment-provider-x service call
	_, paymentProviderSpan := tracing.T.Start(ctx, "paymentProviderXServiceCall")
	// Simulate payment-provider-x processing
	log.Println("Payment provider X service call started")
	time.Sleep(200 * time.Millisecond) // Simulate delay
	log.Println("Payment provider X service call completed")
	paymentProviderSpan.End()

	// Simulate response
	fmt.Fprintln(w, "Payment processed")
	log.Println("Payment processed")
}
