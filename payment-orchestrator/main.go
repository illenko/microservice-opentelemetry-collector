package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/illenko/observability-common"

	"github.com/illenko/payment-orchestrator/handler"
	"github.com/illenko/payment-orchestrator/service"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx := context.Background()

	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	os.Setenv("OTEL_SERVICE_NAME", "payment-orchestrator")

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

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	restyClient := resty.New()
	restyClient.SetDebug(true)
	restyClient.SetTransport(otelhttp.NewTransport(http.DefaultTransport))

	paymentHandler := handler.NewPaymentHandler(service.NewPaymentService(client, restyClient))

	mux := http.NewServeMux()

	handleHTTP(mux, "POST /payments", paymentHandler.Payment)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleHTTP(mux *http.ServeMux, route string, handleFn http.HandlerFunc) {
	instrumentedHandler := otelhttp.NewHandler(otelhttp.WithRouteTag(route, handleFn), route)
	mux.Handle(route, instrumentedHandler)
}
