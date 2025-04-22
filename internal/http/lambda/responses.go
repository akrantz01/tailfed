package lambda

import (
	"encoding/json"
	"net/http"

	"github.com/akrantz01/tailfed/internal/types"
	"github.com/aws/aws-lambda-go/events"
)

// Success creates a successful HTTP response with some data
func Success[T any](data *T) *events.APIGatewayProxyResponse {
	return makeJsonResponse(&types.Response[T]{
		Success: true,
		Data:    data,
	}, http.StatusOK)
}

// InternalServerError creates an error HTTP response for unexpected server errors
func InternalServerError() *events.APIGatewayProxyResponse {
	return Error("internal server error", http.StatusInternalServerError)
}

// Error creates an error HTTP response with a message and status code
func Error(message string, statusCode int) *events.APIGatewayProxyResponse {
	return makeJsonResponse(&types.Response[struct{}]{
		Success: false,
		Error:   message,
	}, statusCode)
}

func makeJsonResponse[T any](body T, statusCode int) *events.APIGatewayProxyResponse {
	encoded, _ := json.Marshal(body)
	return &events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(encoded),
	}
}
