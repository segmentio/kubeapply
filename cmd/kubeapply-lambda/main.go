package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/segmentio/kubeapply/pkg/cluster"
	kaevents "github.com/segmentio/kubeapply/pkg/events"
	"github.com/segmentio/kubeapply/pkg/pullreq"
	"github.com/segmentio/kubeapply/pkg/stats"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
)

var (
	sess        *session.Session
	statsClient stats.StatsClient

	automerge   bool
	debug       bool
	strictCheck bool

	logsURL = getLogsURL()
)

// Lambda parameters passed in through environment variables.
var (
	// Whether this instance should look for the end of successful applies and then
	// automerge. Generally "true" in production and otherwise "false".
	//
	// Optional, defaults to false.
	automergeStr = os.Getenv("KUBEAPPLY_AUTOMERGE")

	// An SSM parameter where a Datadog API key is stored.
	//
	// Optional, defaults to "" (don't export stats to Datadog)
	datadogAPIKeySSMParam = os.Getenv("KUBEAPPLY_DATADOG_API_KEY_SSM_PARAM")

	// Whether to enable debug-level logging.
	//
	// Optional, defaults to false.
	debugStr = os.Getenv("KUBEAPPLY_DEBUG")

	// Environment that the lambda will run in. Only changes in matching clusters
	// be considered.
	//
	// Optional, if blank then changes for all clusters will be considered.
	env = os.Getenv("KUBEAPPLY_ENV")

	// An SSM parameter where a raw github token is stored.
	githubTokenSSMParam = os.Getenv("KUBEAPPLY_GITHUB_TOKEN_SSM_PARAM")

	// Github app key SSM parameter. Either this or the githubTokenSSMParam value previously
	// must be set.
	githubAppKeySSMParam = os.Getenv("KUBEAPPLY_GITHUB_APP_KEY_SSM_PARAM")

	// ID of the app if using app to generate access tokens. Must be set if using app-based
	// authentication.
	githubAppID = os.Getenv("KUBEAPPLY_GITHUB_APP_ID")

	// Installation ID of the app in the organiztion. Must be set if using app-based
	// authentication.
	githubAppInstallationID = os.Getenv("KUBEAPPLY_GITHUB_APP_INSTALLATION_ID")

	// Whether to apply a strict check on the pull request status, approval,
	// and commit lag before allowing an apply. Generally "true" in production
	// and otherwise "false".
	//
	// Optional, defaults to false.
	strictCheckStr = os.Getenv("KUBEAPPLY_STRICT_CHECK")

	// SSM parameter used for fetching webhook secret.
	webhookSecretSSMParam = os.Getenv("KUBEAPPLY_WEBHOOK_SECRET_SSM_PARAM")
)

// Final, decrypted secrets
var (
	githubAccessToken string
	webhookSecret     string
)

func init() {
	sess = session.Must(session.NewSession())

	var err error
	ctx := context.Background()

	if datadogAPIKeySSMParam != "" {
		datadogAPIKey, err := util.GetSSMValue(ctx, sess, datadogAPIKeySSMParam)
		if err != nil {
			log.Fatalf("Error getting datadog key: %+v", err)
		}

		statsClient = stats.NewDatadogStatsClient(
			"kubeapply_lambda.",
			[]string{
				fmt.Sprintf("env:%s", env),
			},
			"kubeapply-lambda",
			datadogAPIKey,
		)
	} else {
		statsClient = &stats.NullStatsClient{}
	}

	if githubTokenSSMParam != "" {
		log.Infof("Getting github access token from ssm")
		githubAccessToken, err = util.GetSSMValue(ctx, sess, githubTokenSSMParam)
	} else if githubAppKeySSMParam != "" {
		log.Infof("Deriving access token from app params")

		githubAppKey, err := util.GetSSMValue(ctx, sess, githubAppKeySSMParam)
		if err != nil {
			log.Fatalf("Error getting github app key: %+v", err)
		}

		jwt, err := pullreq.GenerateJWT(
			githubAppKey,
			githubAppID,
		)
		if err != nil {
			log.Fatalf("Error generating jwt: %+v", err)
		}

		appAccessToken, err := pullreq.GenerateAccessToken(
			ctx,
			jwt,
			githubAppInstallationID,
		)
		if err != nil {
			log.Fatalf("Error generating app access token: %+v", err)
		}
		githubAccessToken = appAccessToken.Token
	} else {
		log.Fatalf("No github token or app key information provided")
	}

	webhookSecret, err = util.GetSSMValue(ctx, sess, webhookSecretSSMParam)
	if err != nil {
		panic(err)
	}

	if strings.ToLower(debugStr) == "true" {
		debug = true
	}

	if strings.ToLower(strictCheckStr) == "true" {
		strictCheck = true
	}

	if strings.ToLower(automergeStr) == "true" {
		automerge = true
	}
}

// Handle handles the lambda invocation and returns a response for the ALB to pass back to
// the client.
func Handle(
	ctx context.Context,
	request events.ALBTargetGroupRequest,
) (events.ALBTargetGroupResponse, error) {
	log.Infof("Got request: %+v", request)

	return handleRequest(ctx, request)
}

func handleRequest(
	ctx context.Context,
	request events.ALBTargetGroupRequest,
) (events.ALBTargetGroupResponse, error) {
	bodyBytes := []byte(request.Body)

	err := kaevents.ValidateSignatureLambdaHeaders(
		request.Headers,
		bodyBytes,
		webhookSecret,
	)
	if err != nil {
		statsClient.Update(
			[]string{"forbidden"},
			[]float64{1.0},
			[]string{},
			stats.StatTypeCount,
		)

		return kaevents.ForbiddenResponse(), nil
	}

	webhookType := kaevents.GetWebhookTypeLambdaHeaders(request.Headers)

	statsClient.Update(
		[]string{"invoked"},
		[]float64{1.0},
		[]string{
			fmt.Sprintf("hook_type:%s", webhookType),
		},
		stats.StatTypeCount,
	)

	webhookContext, err := kaevents.NewWebhookContext(
		webhookType,
		bodyBytes,
		githubAccessToken,
	)
	if err != nil {
		return kaevents.ErrorResponse(err), nil
	} else if webhookContext == nil {
		return kaevents.OKResponse("Not responding"), nil
	}
	defer webhookContext.Close()

	webhookHandler := kaevents.NewWebhookHandler(
		statsClient,
		cluster.NewKubeClusterClient,
		kaevents.WebhookHandlerSettings{
			LogsURL:               logsURL,
			Env:                   env,
			Version:               version.Version,
			StrictCheck:           strictCheck,
			Automerge:             automerge,
			UseLocks:              true,
			ApplyConsistencyCheck: false,
			Debug:                 debug,
		},
	)
	resp := webhookHandler.HandleWebhook(
		ctx,
		webhookContext,
	)
	return resp, nil
}

func getLogsURL() string {
	// See https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html for details
	// on lambda environment variables
	region := os.Getenv("AWS_REGION")
	groupName := os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME")
	streamName := os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME")

	return fmt.Sprintf(
		"https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logEventViewer:group=%s;stream=%s",
		region,
		region,
		groupName,
		streamName,
	)
}

func main() {
	lambda.Start(Handle)
}
