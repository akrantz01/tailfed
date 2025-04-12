package gateway

import (
	"fmt"
	"io"
	"net/http"

	"github.com/akrantz01/tailfed/internal/http/requestid"
	"github.com/aws/aws-lambda-go/events"
)

const (
	ID         = "mock-api"
	DomainName = ID + ".execute-api.us-east-1.amazonaws.com"
	BaseUrl    = "https://" + DomainName
)

// FromHttpRequest creates a new API gateway request from a standard library HTTP request. AWS-specific values are
// filled with dummy values.
func FromHttpRequest(r *http.Request) (events.APIGatewayProxyRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return events.APIGatewayProxyRequest{}, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	queryParams := fromMultiValueMap(r.URL.Query())
	headers := fromMultiValueMap(r.Header)

	return events.APIGatewayProxyRequest{
		Resource:   r.URL.Path,
		Path:       r.URL.Path,
		HTTPMethod: r.Method,

		Headers:           headers,
		MultiValueHeaders: r.Header,

		QueryStringParameters:           queryParams,
		MultiValueQueryStringParameters: r.URL.Query(),

		PathParameters: make(map[string]string),
		StageVariables: make(map[string]string),

		Body:            string(body),
		IsBase64Encoded: false,

		RequestContext: events.APIGatewayProxyRequestContext{
			AccountID:  "111122223333",
			APIID:      ID,
			Stage:      "dev",
			ResourceID: "mock-resource",
			RequestID:  requestid.Get(r),

			DomainName:   DomainName,
			ResourcePath: r.URL.Path,
			HTTPMethod:   r.Method,

			Identity: events.APIGatewayRequestIdentity{
				SourceIP:  r.RemoteAddr,
				UserAgent: r.UserAgent(),
			},
		},
	}, nil
}

func fromMultiValueMap[K comparable, V any](m map[K][]V) map[K]V {
	result := make(map[K]V, len(m))
	for key, values := range m {
		result[key] = values[0]
	}
	return result
}
