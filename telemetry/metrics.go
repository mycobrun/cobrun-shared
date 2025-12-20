// Package telemetry provides observability utilities.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string // OTLP endpoint
	Insecure       bool   // Use insecure connection
}

// MetricsProvider provides metrics functionality.
type MetricsProvider struct {
	provider *sdkmetric.MeterProvider
	meter    metric.Meter
	config   MetricsConfig
}

// NewMetricsProvider creates a new metrics provider.
func NewMetricsProvider(ctx context.Context, config MetricsConfig) (*MetricsProvider, error) {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create meter provider with periodic reader (no-op for now, can be configured with exporters)
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
	)

	// Set global provider
	otel.SetMeterProvider(provider)

	// Get meter
	meter := provider.Meter(config.ServiceName)

	return &MetricsProvider{
		provider: provider,
		meter:    meter,
		config:   config,
	}, nil
}

// Meter returns the meter for creating instruments.
func (m *MetricsProvider) Meter() metric.Meter {
	return m.meter
}

// Shutdown shuts down the metrics provider.
func (m *MetricsProvider) Shutdown(ctx context.Context) error {
	return m.provider.Shutdown(ctx)
}

// Common metrics for rideshare platform

// HTTPMetrics provides HTTP-related metrics.
type HTTPMetrics struct {
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
	requestSize     metric.Int64Histogram
	responseSize    metric.Int64Histogram
	activeRequests  metric.Int64UpDownCounter
}

// NewHTTPMetrics creates HTTP metrics.
func NewHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
	requestsTotal, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, err
	}

	requestSize, err := meter.Int64Histogram(
		"http_request_size_bytes",
		metric.WithDescription("HTTP request size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	responseSize, err := meter.Int64Histogram(
		"http_response_size_bytes",
		metric.WithDescription("HTTP response size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http_active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, err
	}

	return &HTTPMetrics{
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
		activeRequests:  activeRequests,
	}, nil
}

// RecordRequest records HTTP request metrics.
func (m *HTTPMetrics) RecordRequest(ctx context.Context, method, path string, status int, duration time.Duration, reqSize, respSize int64) {
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status_code", status),
		attribute.String("status_class", statusClass(status)),
	}

	m.requestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.requestSize.Record(ctx, reqSize, metric.WithAttributes(attrs...))
	m.responseSize.Record(ctx, respSize, metric.WithAttributes(attrs...))
}

// IncrementActiveRequests increments active requests.
func (m *HTTPMetrics) IncrementActiveRequests(ctx context.Context) {
	m.activeRequests.Add(ctx, 1)
}

// DecrementActiveRequests decrements active requests.
func (m *HTTPMetrics) DecrementActiveRequests(ctx context.Context) {
	m.activeRequests.Add(ctx, -1)
}

func statusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// DatabaseMetrics provides database-related metrics.
type DatabaseMetrics struct {
	operationsTotal    metric.Int64Counter
	operationDuration  metric.Float64Histogram
	connectionPoolSize metric.Int64UpDownCounter
	errorsTotal        metric.Int64Counter
}

// NewDatabaseMetrics creates database metrics.
func NewDatabaseMetrics(meter metric.Meter, dbType string) (*DatabaseMetrics, error) {
	prefix := fmt.Sprintf("db_%s", dbType)

	operationsTotal, err := meter.Int64Counter(
		prefix+"_operations_total",
		metric.WithDescription("Total database operations"),
		metric.WithUnit("{operations}"),
	)
	if err != nil {
		return nil, err
	}

	operationDuration, err := meter.Float64Histogram(
		prefix+"_operation_duration_seconds",
		metric.WithDescription("Database operation duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5),
	)
	if err != nil {
		return nil, err
	}

	connectionPoolSize, err := meter.Int64UpDownCounter(
		prefix+"_connection_pool_size",
		metric.WithDescription("Current database connection pool size"),
		metric.WithUnit("{connections}"),
	)
	if err != nil {
		return nil, err
	}

	errorsTotal, err := meter.Int64Counter(
		prefix+"_errors_total",
		metric.WithDescription("Total database errors"),
		metric.WithUnit("{errors}"),
	)
	if err != nil {
		return nil, err
	}

	return &DatabaseMetrics{
		operationsTotal:    operationsTotal,
		operationDuration:  operationDuration,
		connectionPoolSize: connectionPoolSize,
		errorsTotal:        errorsTotal,
	}, nil
}

// RecordOperation records a database operation.
func (m *DatabaseMetrics) RecordOperation(ctx context.Context, operation string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}

	m.operationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.operationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		m.errorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordPoolSize records the connection pool size.
func (m *DatabaseMetrics) RecordPoolSize(ctx context.Context, size int64) {
	m.connectionPoolSize.Add(ctx, size)
}

// BusinessMetrics provides business-specific metrics for rideshare.
type BusinessMetrics struct {
	tripsRequested   metric.Int64Counter
	tripsCompleted   metric.Int64Counter
	tripsCancelled   metric.Int64Counter
	tripDuration     metric.Float64Histogram
	tripDistance     metric.Float64Histogram
	tripFare         metric.Float64Histogram
	driversOnline    metric.Int64UpDownCounter
	ridersActive     metric.Int64UpDownCounter
	surgeMultiplier  metric.Float64Histogram
	etaAccuracy      metric.Float64Histogram
}

// NewBusinessMetrics creates business metrics.
func NewBusinessMetrics(meter metric.Meter) (*BusinessMetrics, error) {
	tripsRequested, err := meter.Int64Counter(
		"trips_requested_total",
		metric.WithDescription("Total trip requests"),
	)
	if err != nil {
		return nil, err
	}

	tripsCompleted, err := meter.Int64Counter(
		"trips_completed_total",
		metric.WithDescription("Total completed trips"),
	)
	if err != nil {
		return nil, err
	}

	tripsCancelled, err := meter.Int64Counter(
		"trips_cancelled_total",
		metric.WithDescription("Total cancelled trips"),
	)
	if err != nil {
		return nil, err
	}

	tripDuration, err := meter.Float64Histogram(
		"trip_duration_minutes",
		metric.WithDescription("Trip duration in minutes"),
		metric.WithUnit("min"),
		metric.WithExplicitBucketBoundaries(5, 10, 15, 20, 30, 45, 60, 90, 120),
	)
	if err != nil {
		return nil, err
	}

	tripDistance, err := meter.Float64Histogram(
		"trip_distance_km",
		metric.WithDescription("Trip distance in kilometers"),
		metric.WithUnit("km"),
		metric.WithExplicitBucketBoundaries(1, 2, 5, 10, 15, 20, 30, 50, 100),
	)
	if err != nil {
		return nil, err
	}

	tripFare, err := meter.Float64Histogram(
		"trip_fare_usd",
		metric.WithDescription("Trip fare in USD"),
		metric.WithUnit("USD"),
		metric.WithExplicitBucketBoundaries(5, 10, 15, 20, 30, 50, 75, 100, 150, 200),
	)
	if err != nil {
		return nil, err
	}

	driversOnline, err := meter.Int64UpDownCounter(
		"drivers_online",
		metric.WithDescription("Number of online drivers"),
	)
	if err != nil {
		return nil, err
	}

	ridersActive, err := meter.Int64UpDownCounter(
		"riders_active",
		metric.WithDescription("Number of riders with active trips"),
	)
	if err != nil {
		return nil, err
	}

	surgeMultiplier, err := meter.Float64Histogram(
		"surge_multiplier",
		metric.WithDescription("Current surge pricing multiplier"),
		metric.WithExplicitBucketBoundaries(1.0, 1.25, 1.5, 1.75, 2.0, 2.5, 3.0, 4.0, 5.0),
	)
	if err != nil {
		return nil, err
	}

	etaAccuracy, err := meter.Float64Histogram(
		"eta_accuracy_seconds",
		metric.WithDescription("ETA accuracy (actual - predicted) in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &BusinessMetrics{
		tripsRequested:   tripsRequested,
		tripsCompleted:   tripsCompleted,
		tripsCancelled:   tripsCancelled,
		tripDuration:     tripDuration,
		tripDistance:     tripDistance,
		tripFare:         tripFare,
		driversOnline:    driversOnline,
		ridersActive:     ridersActive,
		surgeMultiplier:  surgeMultiplier,
		etaAccuracy:      etaAccuracy,
	}, nil
}

// RecordTripRequested records a trip request.
func (m *BusinessMetrics) RecordTripRequested(ctx context.Context, rideType, city string) {
	attrs := []attribute.KeyValue{
		attribute.String("ride_type", rideType),
		attribute.String("city", city),
	}
	m.tripsRequested.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordTripCompleted records a completed trip.
func (m *BusinessMetrics) RecordTripCompleted(ctx context.Context, rideType, city string, durationMin, distanceKm, fareUSD float64) {
	attrs := []attribute.KeyValue{
		attribute.String("ride_type", rideType),
		attribute.String("city", city),
	}
	m.tripsCompleted.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.tripDuration.Record(ctx, durationMin, metric.WithAttributes(attrs...))
	m.tripDistance.Record(ctx, distanceKm, metric.WithAttributes(attrs...))
	m.tripFare.Record(ctx, fareUSD, metric.WithAttributes(attrs...))
}

// RecordTripCancelled records a cancelled trip.
func (m *BusinessMetrics) RecordTripCancelled(ctx context.Context, rideType, city, reason string) {
	attrs := []attribute.KeyValue{
		attribute.String("ride_type", rideType),
		attribute.String("city", city),
		attribute.String("reason", reason),
	}
	m.tripsCancelled.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordDriverOnline records a driver coming online.
func (m *BusinessMetrics) RecordDriverOnline(ctx context.Context, city string) {
	attrs := []attribute.KeyValue{
		attribute.String("city", city),
	}
	m.driversOnline.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordDriverOffline records a driver going offline.
func (m *BusinessMetrics) RecordDriverOffline(ctx context.Context, city string) {
	attrs := []attribute.KeyValue{
		attribute.String("city", city),
	}
	m.driversOnline.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// RecordSurgeMultiplier records the current surge multiplier.
func (m *BusinessMetrics) RecordSurgeMultiplier(ctx context.Context, city string, multiplier float64) {
	attrs := []attribute.KeyValue{
		attribute.String("city", city),
	}
	m.surgeMultiplier.Record(ctx, multiplier, metric.WithAttributes(attrs...))
}

// RecordETAAccuracy records ETA accuracy.
func (m *BusinessMetrics) RecordETAAccuracy(ctx context.Context, actualSeconds, predictedSeconds float64) {
	diff := actualSeconds - predictedSeconds
	m.etaAccuracy.Record(ctx, diff)
}

// MetricsMiddleware creates an HTTP middleware that records metrics.
func MetricsMiddleware(metrics *HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			metrics.IncrementActiveRequests(ctx)
			defer metrics.DecrementActiveRequests(ctx)

			start := time.Now()

			// Wrap response writer to capture status and size
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			metrics.RecordRequest(
				ctx,
				r.Method,
				r.URL.Path,
				wrapped.status,
				duration,
				r.ContentLength,
				int64(wrapped.size),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}
