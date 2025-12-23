package telemetry

import (
	"context"
	"testing"
)

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		name                   string
		connStr                string
		wantInstrumentationKey string
		wantIngestionEndpoint  string
	}{
		{
			name:                   "valid connection string",
			connStr:                "InstrumentationKey=abc123;IngestionEndpoint=https://eastus-0.in.applicationinsights.azure.com/",
			wantInstrumentationKey: "abc123",
			wantIngestionEndpoint:  "https://eastus-0.in.applicationinsights.azure.com",
		},
		{
			name:                   "with live endpoint",
			connStr:                "InstrumentationKey=xyz789;IngestionEndpoint=https://westus2.in.applicationinsights.azure.com/;LiveEndpoint=https://westus2.livediagnostics.monitor.azure.com/",
			wantInstrumentationKey: "xyz789",
			wantIngestionEndpoint:  "https://westus2.in.applicationinsights.azure.com",
		},
		{
			name:                   "missing ingestion endpoint",
			connStr:                "InstrumentationKey=test123",
			wantInstrumentationKey: "test123",
			wantIngestionEndpoint:  "",
		},
		{
			name:                   "empty string",
			connStr:                "",
			wantInstrumentationKey: "",
			wantIngestionEndpoint:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotEndpoint := parseConnectionString(tt.connStr)
			if gotKey != tt.wantInstrumentationKey {
				t.Errorf("instrumentationKey = %q, want %q", gotKey, tt.wantInstrumentationKey)
			}
			if gotEndpoint != tt.wantIngestionEndpoint {
				t.Errorf("ingestionEndpoint = %q, want %q", gotEndpoint, tt.wantIngestionEndpoint)
			}
		})
	}
}

func TestDefaultAzureConfig(t *testing.T) {
	config := DefaultAzureConfig()

	if config.SampleRate != 1.0 {
		t.Errorf("SampleRate = %f, want 1.0", config.SampleRate)
	}

	if config.MetricExportInterval.Seconds() != 60 {
		t.Errorf("MetricExportInterval = %v, want 60s", config.MetricExportInterval)
	}
}

func TestNewAzureTelemetry_MissingKey(t *testing.T) {
	ctx := context.Background()
	config := AzureConfig{
		ConnectionString: "IngestionEndpoint=https://test.azure.com/",
		ServiceName:      "test-service",
	}

	_, err := NewAzureTelemetry(ctx, config)
	if err == nil {
		t.Error("expected error for missing instrumentation key")
	}
}

func TestGetHostname(t *testing.T) {
	hostname := getHostname()
	if hostname == "" {
		t.Error("expected non-empty hostname")
	}
}

func TestMustInitFromEnv_NotConfigured(t *testing.T) {
	// Should not panic when not configured
	ctx := context.Background()
	tel := MustInitFromEnv(ctx)
	if tel != nil {
		tel.Shutdown(ctx)
	}
}
