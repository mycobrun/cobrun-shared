// Package maps provides a Google Maps Platform adapter.
package maps

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer wraps an OpenTelemetry tracer for maps operations.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new Tracer wrapping an OpenTelemetry tracer.
func NewTracer(tracer trace.Tracer) *Tracer {
	if tracer == nil {
		return nil
	}
	return &Tracer{tracer: tracer}
}

// Span wraps an OpenTelemetry span.
type Span struct {
	span trace.Span
}

// End ends the span.
func (s *Span) End() {
	if s.span != nil {
		s.span.End()
	}
}

// RecordError records an error on the span.
func (s *Span) RecordError(err error) {
	if s.span != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// SetAttributes sets attributes on the span.
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	if s.span != nil {
		s.span.SetAttributes(attrs...)
	}
}

// StartSpan starts a new span for maps operations.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if t == nil || t.tracer == nil {
		return ctx, &Span{}
	}

	ctx, span := t.tracer.Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("maps.provider", "google"),
		),
	)

	return ctx, &Span{span: span}
}

// MapsAttributes returns common attributes for maps operations.
func MapsAttributes(operation string, originLat, originLng, destLat, destLng float64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("maps.operation", operation),
		attribute.Float64("maps.origin.lat", originLat),
		attribute.Float64("maps.origin.lng", originLng),
		attribute.Float64("maps.dest.lat", destLat),
		attribute.Float64("maps.dest.lng", destLng),
	}
}

// AutocompleteAttributes returns attributes for autocomplete operations.
func AutocompleteAttributes(input string, resultsCount int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("maps.operation", "autocomplete"),
		attribute.Int("maps.input.length", len(input)),
		attribute.Int("maps.results.count", resultsCount),
	}
}

// RouteAttributes returns attributes for route operations.
func RouteAttributes(distanceMeters, durationSeconds int, trafficDelay int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("maps.operation", "compute_routes"),
		attribute.Int("maps.distance.meters", distanceMeters),
		attribute.Int("maps.duration.seconds", durationSeconds),
		attribute.Int("maps.traffic.delay_seconds", trafficDelay),
	}
}

// MatrixAttributes returns attributes for matrix operations.
func MatrixAttributes(originsCount, destinationsCount, elementsCount int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("maps.operation", "compute_matrix"),
		attribute.Int("maps.matrix.origins", originsCount),
		attribute.Int("maps.matrix.destinations", destinationsCount),
		attribute.Int("maps.matrix.elements", elementsCount),
	}
}

// GeocodeAttributes returns attributes for geocode operations.
func GeocodeAttributes(lat, lng float64, resultCity string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("maps.operation", "reverse_geocode"),
		attribute.Float64("maps.location.lat", lat),
		attribute.Float64("maps.location.lng", lng),
		attribute.String("maps.result.city", resultCity),
	}
}
