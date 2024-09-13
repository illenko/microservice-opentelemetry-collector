package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/illenko/observability-common"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RoutingResponse struct {
	ID              string `json:"id"`
	PaymentProvider string `json:"paymentProvider"`
	URL             string `json:"url"`
}

func main() {
	ctx := context.Background()

	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	os.Setenv("OTEL_SERVICE_NAME", "routing-service")

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

	mux := http.NewServeMux()
	handleHTTP(mux, "GET /routings/{id}", routingHandler)
	log.Fatal(http.ListenAndServe(":8081", mux))
}

func handleHTTP(mux *http.ServeMux, route string, handleFn http.HandlerFunc) {
	instrumentedHandler := otelhttp.NewHandler(otelhttp.WithRouteTag(route, handleFn), route)
	mux.Handle(route, instrumentedHandler)
}

func routingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")

	response := RoutingResponse{
		ID:              id,
		PaymentProvider: "pay-x",
		URL:             "http://localhost:8082/pay",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(ctx, "Failed to encode response", slog.Any("error", err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "Routing request processed", slog.String("ID", id))
}
