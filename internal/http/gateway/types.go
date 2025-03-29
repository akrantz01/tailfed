package gateway

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// Handler processes requests from an AWS API gateway
type Handler interface {
	Serve(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)
}
