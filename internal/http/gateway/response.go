package gateway

import (
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// WriteHttpResponse writes an API gateway response to a HTTP response
func WriteHttpResponse(w http.ResponseWriter, resp *events.APIGatewayProxyResponse) error {
	headers := w.Header()
	for key, value := range resp.Headers {
		headers.Set(key, value)
	}
	for key, values := range resp.MultiValueHeaders {
		headers.Del(key)
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if resp.IsBase64Encoded {
		return errors.New("base64-encoded bodies are unsupported")
	}

	_, err := w.Write([]byte(resp.Body))
	return err
}
