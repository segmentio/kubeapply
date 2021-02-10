package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"reflect"

	"github.com/gorilla/mux"
	"github.com/segmentio/conf"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/events"
	kstats "github.com/segmentio/kubeapply/pkg/stats"
	"github.com/segmentio/kubeapply/pkg/version"
	"github.com/segmentio/stats/httpstats"
	"github.com/segmentio/stats/v4"
	"github.com/segmentio/stats/v4/datadog"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"k8s.io/klog/v2"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	klog.SetOutput(os.Stderr)
}

// Config is used to configure the webhooks server. The parameters can be set via either
// environment variables or flags. See https://github.com/segmentio/conf for more details.
//
// TODO: Support Github app credentials in addition to account tokens.
type Config struct {
	Automerge     bool   `conf:"automerge"      help:"automerge changes after successful apply"`
	Bind          string `conf:"bind"           help:"binding address"`
	Debug         bool   `conf:"debug"          help:"turn on debug logging"`
	DogStatsdAddr string `conf:"dogstatsd-addr" help:"address for datadog-formatted statsd metrics"`
	Env           string `conf:"env"            help:"only consider changes for this environment"`
	GithubToken   string `conf:"github-token"   help:"token for Github API access"`
	LogsURL       string `conf:"logs-url"       help:"url for logs; used as link for status checks"`
	WebhookSecret string `conf:"webhook-secret" help:"shared secret set in Github webhooks"`

	// TODO: Deprecate StrictCheck since it's covered by the parameters below that.
	StrictCheck     bool `conf:"strict-check"      help:"ensure green status and approval before apply"`
	GreenCIRequired bool `conf:"green-ci-required" help:"require green CI before applying"`
	ReviewRequired  bool `conf:"review-required"   help:"require review before applying:"`
}

var config = Config{
	Bind: ":8080",
}

func main() {
	conf.Load(&config)

	if config.DogStatsdAddr != "" {
		datadogClient := datadog.NewClient(config.DogStatsdAddr)
		stats.Register(datadogClient)
		defer stats.Flush()
	}

	router := mux.NewRouter()
	router.HandleFunc("/webhook", webhookHTTPHandler).Methods("POST")

	server := &http.Server{
		Handler: httpstats.NewHandler(router),
		Addr:    config.Bind,
	}

	log.Infof("Starting server on %s", config.Bind)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error running server: %+v", err)
	}
}

func webhookHTTPHandler(
	writer http.ResponseWriter,
	req *http.Request,
) {
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		respondWithError(writer, req, 500, err)
		return
	}
	defer req.Body.Close()

	err = events.ValidateSignatureHTTPHeaders(
		req.Header,
		bodyBytes,
		config.WebhookSecret,
	)
	if err != nil {
		respondWithError(writer, req, 403, err)
		return
	}

	webhookType := events.GetWebhookTypeHTTPHeaders(req.Header)

	webhookContext, err := events.NewWebhookContext(
		webhookType,
		bodyBytes,
		config.GithubToken,
	)
	if err != nil {
		respondWithError(writer, req, 500, err)
		return
	} else if webhookContext == nil {
		respondWithText(writer, req, 200, "Non-matching event")
		return
	}
	defer webhookContext.Close()

	webhookHandler := events.NewWebhookHandler(
		kstats.NewSegmentStatsClient(stats.DefaultEngine),
		cluster.NewKubeClusterClient,
		events.WebhookHandlerSettings{
			LogsURL:               config.LogsURL,
			Env:                   config.Env,
			Version:               version.Version,
			UseLocks:              true,
			ApplyConsistencyCheck: false,
			Automerge:             config.Automerge,
			StrictCheck:           config.StrictCheck,
			GreenCIRequired:       config.GreenCIRequired,
			ReviewRequired:        config.ReviewRequired,
			Debug:                 config.Debug,
		},
	)
	response := webhookHandler.HandleWebhook(req.Context(), webhookContext)
	log.Infof("Webhook response: %+v", response)

	writer.Header().Set("Content-Type", "text/plain")
	writer.WriteHeader(response.StatusCode)
	for key, value := range response.Headers {
		writer.Header().Add(key, value)
	}
	writer.Write([]byte(response.Body))
}

func respondWithText(
	writer http.ResponseWriter,
	req *http.Request,
	code int,
	text string,
) {
	writer.Header().Set("Content-Type", "text/plain")
	writer.WriteHeader(code)
	writer.Write([]byte(text))
}

func respondWithError(
	writer http.ResponseWriter,
	req *http.Request,
	code int,
	err error,
) {
	log.Warnf(
		"Error for URI %s [%d]: %+v (type:%s)",
		req.RequestURI,
		code,
		err,
		reflect.TypeOf(err).String(),
	)
	respondWithText(writer, req, code, err.Error())
}
