package events

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	log "github.com/sirupsen/logrus"
)

// OKResponse returns a 200 ALB response with the provided body.
func OKResponse(body string) events.ALBTargetGroupResponse {
	log.Infof("Returning OK response with body: %s", body)
	return events.ALBTargetGroupResponse{
		Body:              body,
		StatusCode:        200,
		StatusDescription: "200 OK",
		IsBase64Encoded:   false,
		Headers:           map[string]string{},
	}
}

// ForbiddenResponse returns a 403 ALB response.
func ForbiddenResponse() events.ALBTargetGroupResponse {
	log.Warnf("Returning forbidden response")
	return events.ALBTargetGroupResponse{
		Body:              "Forbidden",
		StatusCode:        403,
		StatusDescription: "403 Forbidden",
		IsBase64Encoded:   false,
		Headers:           map[string]string{},
	}
}

// ErrorResponse returns a 500 ALB response with a body generated from the provided error.
func ErrorResponse(err error) events.ALBTargetGroupResponse {
	log.Warnf("Returning error response with err: %+v", err)
	return events.ALBTargetGroupResponse{
		Body:              fmt.Sprintf("Error: %+v", err),
		StatusCode:        500,
		StatusDescription: "500 Internal Server Error",
		IsBase64Encoded:   false,
		Headers:           map[string]string{},
	}
}
