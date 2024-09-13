package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/illenko/observability-common"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RoutingResponse struct {
	ID              string `json:"id"`
	PaymentProvider string `json:"paymentProvider"`
	URL             string `json:"url"`
}

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		slog.ErrorContext(ctx, "Error loading .env file", slog.Any("error", err))
		return
	}

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
	slog.ErrorContext(ctx, "server failed", slog.Any("error", http.ListenAndServe(":"+os.Getenv("PORT"), mux)))
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
