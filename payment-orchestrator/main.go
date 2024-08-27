package main

import (
	"context"
	"log"
	"net/http"

	"github.com/illenko/observability-common/tracing"
	"github.com/illenko/payment-orchestrator/handler"
	"github.com/illenko/payment-orchestrator/service"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	ctx := context.Background()

	tp, _, err := tracing.InitProvider("localhost:4317", "payment-orchestrator", ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	restyClient := resty.New()
	restyClient.SetDebug(true)

	paymentHandler := handler.NewPaymentHandler(service.NewPaymentService(client, restyClient))

	mux := http.NewServeMux()
	mux.Handle("POST /payments", otelhttp.NewHandler(http.HandlerFunc(paymentHandler.Payment), "payment", otelhttp.WithPropagators(propagation.TraceContext{})))
	log.Fatal(http.ListenAndServe(":8080", mux))
}
