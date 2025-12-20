module github.com/cobrun/cobrun-platform/pkg

go 1.22.0

require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.5.1
	github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos v0.3.6
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus v1.6.0
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs v1.0.3
	github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.1.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-playground/validator/v10 v10.18.0
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.6.0
	github.com/microsoft/ApplicationInsights-Go v0.4.4
	github.com/redis/go-redis/v9 v9.5.1
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0
)
