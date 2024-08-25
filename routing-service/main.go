package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"

	"github.com/illenko/observability-common/tracing"
)

type RoutingResponse struct {
	ID              string `json:"id"`
	PaymentProvider string `json:"paymentProvider"`
	URL             string `json:"url"`
}

func main() {
	ctx := context.Background()

	tp, err := tracing.InitProvider("localhost:4317", "routing-service", ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("GET /routings/{id}", otelhttp.NewHandler(http.HandlerFunc(routingHandler), "routing", otelhttp.WithPropagators(propagation.TraceContext{})))
	log.Fatal(http.ListenAndServe(":8081", mux))
}

func routingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")

	ctx, routeSearchSpan := tracing.T.Start(ctx, "routeSearch")

	log.Println("Route search started")
	time.Sleep(150 * time.Millisecond)
	log.Println("Route search completed")
	routeSearchSpan.End()

	response := RoutingResponse{
		ID:              id,
		PaymentProvider: "pay-x",
		URL:             "http://localhost:8082/pay",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("Routing request for ID: %s processed", id)
}
