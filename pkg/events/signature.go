package events

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/go-github/v30/github"
)

const (
	signatureHeader = "X-Hub-Signature"
)

// ValidateSignatureLambdaHeaders validates a github webhook signature assuming lambda-formatted
// headers.
func ValidateSignatureLambdaHeaders(headers map[string]string, body []byte, secret string) error {
	value, ok := headers[strings.ToLower(signatureHeader)]
	if !ok || value == "" {
		return errors.New("signature header not set")
	}

	return validateSignature(
		value,
		body,
		secret,
	)
}

// ValidateSignatureHTTPHeaders validates a github webhook signature assuming http-formatted
// headers.
func ValidateSignatureHTTPHeaders(headers http.Header, body []byte, secret string) error {
	values, ok := headers[signatureHeader]
	if !ok || len(values) == 0 {
		return errors.New("signature header not set")
	}

	return validateSignature(
		values[0],
		body,
		secret,
	)
}

func validateSignature(signature string, body []byte, secret string) error {
	return github.ValidateSignature(
		signature,
		body,
		[]byte(secret),
	)
}
