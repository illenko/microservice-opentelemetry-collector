package main

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"

	"github.com/illenko/observability-common/tracing"
)

var (
	requestCount   metric.Int64Counter
	requestLatency metric.Float64Histogram
	initOnce       sync.Once
)

type Config struct {
	ServiceName string
	Endpoint    string
}

func metricsMiddleware(next http.Handler, serviceName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		initOnce.Do(func() {
			requestCount, _ = tracing.M.Int64Counter("http_request_count")
			requestLatency, _ = tracing.M.Float64Histogram("http_request_latency", metric.WithExplicitBucketBoundaries(0.5, 0.9, 0.95, 0.99))
		})

		next.ServeHTTP(w, r)
	})
}

type RoutingResponse struct {
	ID              string `json:"id"`
	PaymentProvider string `json:"paymentProvider"`
	URL             string `json:"url"`
}

func main() {
	ctx := context.Background()

	tp, _, err := tracing.InitProvider("localhost:4317", "routing-service", ctx)
	if err != nil {
		log.Fatal(err)
	}

	requestCount, _ = tracing.M.Int64Counter("routing_service_request_count")
	requestLatency, _ = tracing.M.Float64Histogram("routing_service_request_latency")

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/routings/{id}", metricsMiddleware(otelhttp.NewHandler(http.HandlerFunc(routingHandler), "routing", otelhttp.WithPropagators(propagation.TraceContext{})), "routing-service"))
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

	startTime := time.Now()
	log.Println("Route search started")
	time.Sleep(150 * time.Millisecond)
	log.Println("Route search completed")
	routeSearchSpan.End()

	requestCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("service.name", "routing-service"),
		attribute.String("http.path", r.URL.Path),
	))
	requestLatency.Record(ctx, time.Since(startTime).Seconds(), metric.WithAttributes(
		attribute.String("service.name", "routing-service"),
		attribute.String("http.path", r.URL.Path),
	))

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
