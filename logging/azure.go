// Package logging provides Azure Application Insights integration.
package logging

import (
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

// AppInsightsClient wraps Application Insights telemetry client.
type AppInsightsClient struct {
	client appinsights.TelemetryClient
}

// NewAppInsightsClient creates a new Application Insights client.
func NewAppInsightsClient(instrumentationKey string) *AppInsightsClient {
	if instrumentationKey == "" {
		return nil
	}

	config := appinsights.NewTelemetryConfiguration(instrumentationKey)
	config.MaxBatchSize = 8192
	config.MaxBatchInterval = 2 * time.Second

	client := appinsights.NewTelemetryClientFromConfig(config)

	return &AppInsightsClient{client: client}
}

// TrackEvent tracks a custom event.
func (c *AppInsightsClient) TrackEvent(name string, properties map[string]string) {
	if c == nil || c.client == nil {
		return
	}
	event := appinsights.NewEventTelemetry(name)
	for k, v := range properties {
		event.Properties[k] = v
	}
	c.client.Track(event)
}

// TrackMetric tracks a custom metric.
func (c *AppInsightsClient) TrackMetric(name string, value float64) {
	if c == nil || c.client == nil {
		return
	}
	metric := appinsights.NewMetricTelemetry(name, value)
	c.client.Track(metric)
}

// TrackException tracks an exception.
func (c *AppInsightsClient) TrackException(err error) {
	if c == nil || c.client == nil {
		return
	}
	exception := appinsights.NewExceptionTelemetry(err)
	c.client.Track(exception)
}

// TrackRequest tracks an HTTP request.
func (c *AppInsightsClient) TrackRequest(name, url string, duration time.Duration, responseCode string, success bool) {
	if c == nil || c.client == nil {
		return
	}
	request := appinsights.NewRequestTelemetry(name, url, duration, responseCode)
	request.Success = success
	c.client.Track(request)
}

// TrackDependency tracks a dependency call (database, external API, etc.).
func (c *AppInsightsClient) TrackDependency(name, dependencyType, target, data string, duration time.Duration, success bool) {
	if c == nil || c.client == nil {
		return
	}
	dependency := appinsights.NewRemoteDependencyTelemetry(name, dependencyType, target, success)
	dependency.Duration = duration
	dependency.Data = data
	c.client.Track(dependency)
}

// Flush flushes all pending telemetry.
func (c *AppInsightsClient) Flush() {
	if c == nil || c.client == nil {
		return
	}
	c.client.Channel().Flush()
}

// Close closes the client and flushes remaining telemetry.
func (c *AppInsightsClient) Close() {
	if c == nil || c.client == nil {
		return
	}
	c.client.Channel().Close()
}
