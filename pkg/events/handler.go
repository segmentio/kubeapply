package events

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/pullreq"
	"github.com/segmentio/kubeapply/pkg/stats"
	log "github.com/sirupsen/logrus"
)

const (
	applyTimeout = 600 * time.Second
	diffTimeout  = 600 * time.Second
)

// WebhookHandler is a struct that handles incoming Github webhooks. Depending on the webhook
// details, it may make changes in one or more Kubernetes clusters, post comments back in the
// pull request, etc.
type WebhookHandler struct {
	statsClient     stats.StatsClient
	clientGenerator cluster.ClusterClientGenerator
	settings        WebhookHandlerSettings
}

// WebhookHandlerSettings stores the settings associated with a WebhookHandler.
type WebhookHandlerSettings struct {
	// ApplyConsistencyCheck indicates whether we should check that the SHA of an apply matches
	// the SHA of the last diff for the cluster.
	ApplyConsistencyCheck bool

	// Automerge indicates whether the handler should automatically merge changes after applies
	// have been made successfully in all clusters.
	Automerge bool

	// Debug indicates whether we should enable debug-level logging on kubectl calls.
	Debug bool

	// Env is the environment for this handler.
	Env string

	// LogsURL is the URL that should be used
	LogsURL string

	// StrictCheck indicates whether we should block applies on having an approval and all
	// green statuses.
	StrictCheck bool

	// UseLocks indicates whether we should use locking to prevent overlapping handler calls
	// for a cluster.
	UseLocks bool

	// Version is the version of the lambda that invokes this handler.
	Version string
}

// NewWebhookHandler creates a new WebhookHandler from the provided clients and settings.
func NewWebhookHandler(
	statsClient stats.StatsClient,
	clientGenerator cluster.ClusterClientGenerator,
	settings WebhookHandlerSettings,
) *WebhookHandler {
	return &WebhookHandler{
		statsClient:     statsClient,
		clientGenerator: clientGenerator,
		settings:        settings,
	}
}

// HandleWebhook processes a single WebhookContext, returning an ALB response that should
// be passed back to the client.
func (whh *WebhookHandler) HandleWebhook(
	ctx context.Context,
	webhookContext *WebhookContext,
) events.ALBTargetGroupResponse {
	if webhookContext == nil {
		return OKResponse("OK")
	} else if webhookContext.pullRequestEvent != nil {
		return whh.handlePullRequestEvent(ctx, webhookContext)
	} else if webhookContext.issueCommentEvent != nil {
		if webhookContext.commentType == commentTypeCommand {
			return whh.handleCommandCommentEvent(ctx, webhookContext)
		}
		return whh.handleApplyResultCommentEvent(ctx, webhookContext)
	} else {
		return ErrorResponse(errors.New("Context missing from event"))
	}
}

func (whh *WebhookHandler) handlePullRequestEvent(
	ctx context.Context,
	webhookContext *WebhookContext,
) events.ALBTargetGroupResponse {
	err := webhookContext.pullRequestClient.Init(ctx)
	if err != nil {
		whh.incrementStat("handler.pull_request.error", webhookContext, "")
		webhookContext.pullRequestClient.PostErrorComment(ctx, whh.settings.Env, err)
		return ErrorResponse(err)
	}

	clusterClients, err := whh.getClusterClients(
		ctx,
		webhookContext.pullRequestClient,
		nil,
		nil,
	)
	if err != nil {
		whh.incrementStat("handler.pull_request.error", webhookContext, "")
		webhookContext.pullRequestClient.PostErrorComment(ctx, whh.settings.Env, err)
		return ErrorResponse(err)
	}
	if len(clusterClients) == 0 {
		return OKResponse("No clusters affected by this change")
	}

	defer func() {
		for _, clusterClient := range clusterClients {
			clusterClient.Close()
		}
	}()

	action := webhookContext.pullRequestEvent.GetAction()

	if action == "opened" {
		// Post help at the beginning
		err := whh.runHelp(ctx, webhookContext.pullRequestClient, clusterClients)
		if err != nil {
			whh.incrementStat("handler.pull_request.error", webhookContext, "help")
			return ErrorResponse(err)
		}

		whh.incrementStat("handler.pull_request.success", webhookContext, "help")
	}

	err = whh.runDiffs(ctx, webhookContext.pullRequestClient, clusterClients)
	if err != nil {
		whh.incrementStat("handler.pull_request.error", webhookContext, "diff")
		return ErrorResponse(err)
	}

	whh.incrementStat("handler.pull_request.success", webhookContext, "diff")

	return OKResponse("OK")
}

func (whh *WebhookHandler) handleCommandCommentEvent(
	ctx context.Context,
	webhookContext *WebhookContext,
) events.ALBTargetGroupResponse {
	err := webhookContext.pullRequestClient.Init(ctx)
	if err != nil {
		whh.incrementStat("handler.comment.error", webhookContext, "")
		webhookContext.pullRequestClient.PostErrorComment(ctx, whh.settings.Env, err)
		return ErrorResponse(err)
	}

	commentBody := webhookContext.issueCommentEvent.GetComment().GetBody()

	eventCommand, err := getCommand(commentBody)
	if err != nil {
		whh.incrementStat("handler.comment.error", webhookContext, "")
		webhookContext.pullRequestClient.PostErrorComment(
			ctx,
			whh.settings.Env,
			errors.New("Sorry, I didn't understand; post \"kubeapply help\" for usage."),
		)
		return ErrorResponse(errors.New("Unrecognized command"))
	}

	clusterClients, err := whh.getClusterClients(
		ctx,
		webhookContext.pullRequestClient,
		eventCommand.args,
		eventCommand.flags,
	)
	if err != nil {
		webhookContext.pullRequestClient.PostErrorComment(ctx, whh.settings.Env, err)
		return ErrorResponse(err)
	}
	if len(clusterClients) == 0 {
		return OKResponse("No clusters affected by this change")
	}

	defer func() {
		for _, clusterClient := range clusterClients {
			clusterClient.Close()
		}
	}()

	switch eventCommand.cmd {
	case commandApply:
		err = whh.runApply(
			ctx,
			webhookContext.pullRequestClient,
			clusterClients,
			eventCommand.flags,
		)
		if err != nil {
			whh.incrementStat("handler.comment.error", webhookContext, "apply")
			return ErrorResponse(err)
		}

		whh.incrementStat("handler.comment.success", webhookContext, "apply")
	case commandDiff:
		err = whh.runDiffs(ctx, webhookContext.pullRequestClient, clusterClients)
		if err != nil {
			whh.incrementStat("handler.comment.error", webhookContext, "diff")
			return ErrorResponse(err)
		}

		whh.incrementStat("handler.comment.success", webhookContext, "diff")
	case commandStatus:
		err = whh.runStatus(ctx, webhookContext.pullRequestClient, clusterClients)
		if err != nil {
			whh.incrementStat("handler.comment.error", webhookContext, "status")
			return ErrorResponse(err)
		}

		whh.incrementStat("handler.comment.success", webhookContext, "status")
	case commandHelp:
		err = whh.runHelp(ctx, webhookContext.pullRequestClient, clusterClients)
		if err != nil {
			whh.incrementStat("handler.comment.error", webhookContext, "help")
			return ErrorResponse(err)
		}

		whh.incrementStat("handler.comment.success", webhookContext, "help")
	}

	return OKResponse("OK")
}

type preMergeCondition struct {
	description string
	value       bool
}

func (whh *WebhookHandler) handleApplyResultCommentEvent(
	ctx context.Context,
	webhookContext *WebhookContext,
) events.ALBTargetGroupResponse {
	if !whh.settings.Automerge {
		log.Infof("Not automerging because automerge setting is set to false")
		return OKResponse("OK")
	}

	err := webhookContext.pullRequestClient.Init(ctx)
	if err != nil {
		whh.incrementStat("handler.automerge.error", webhookContext, "")
		webhookContext.pullRequestClient.PostErrorComment(ctx, whh.settings.Env, err)
		return ErrorResponse(err)
	}

	preMergeConditions := []preMergeCondition{
		{
			description: "all statuses green",
			value:       statusAllGreen(ctx, webhookContext.pullRequestClient),
		},
		{
			description: "workflows completed",
			value:       statusWorkflowCompleted(ctx, webhookContext.pullRequestClient),
		},
		{
			description: "pull request is not a draft",
			value:       !webhookContext.pullRequestClient.IsDraft(ctx),
		},
		{
			description: "pull request is not already merged",
			value:       !webhookContext.pullRequestClient.IsMerged(ctx),
		},
		{
			description: "pull request is mergeble",
			value:       webhookContext.pullRequestClient.IsMergeable(ctx),
		},
	}

	for _, condition := range preMergeConditions {
		if !condition.value {
			log.Warnf(
				"Not automerging because required condition is not true: %s",
				condition.description,
			)
			return OKResponse("OK")
		}
	}

	err = webhookContext.pullRequestClient.PostComment(
		ctx,
		"🎉 Auto-merging because changes have been successfully applied in all clusters 🎉",
	)
	if err != nil {
		return ErrorResponse(err)
	}

	err = webhookContext.pullRequestClient.Merge(ctx)
	if err != nil {
		return ErrorResponse(err)
	}

	whh.incrementStat("handler.automerge.success", webhookContext, "")
	return OKResponse("OK")
}

func (whh *WebhookHandler) getClusterClients(
	ctx context.Context,
	client pullreq.PullRequestClient,
	clusterIDs []string,
	flags map[string]string,
) ([]cluster.ClusterClient, error) {
	clusterClients := []cluster.ClusterClient{}

	coveredClusters, err := client.GetCoveredClusters(
		whh.settings.Env,
		clusterIDs,
		flags["subpath"],
	)
	if err != nil {
		return nil, err
	}
	headSHA := client.HeadSHA()

	for _, coveredCluster := range coveredClusters {
		clusterClient, err := whh.clientGenerator(
			ctx,
			&cluster.ClusterClientConfig{
				ClusterConfig:         coveredCluster,
				HeadSHA:               headSHA,
				CheckApplyConsistency: whh.settings.ApplyConsistencyCheck,
				UseLocks:              whh.settings.UseLocks,
				Debug:                 whh.settings.Debug,
			},
		)
		if err != nil {
			return nil, err
		}

		clusterClients = append(
			clusterClients,
			clusterClient,
		)
	}

	return clusterClients, nil
}

func (whh *WebhookHandler) runApply(
	ctx context.Context,
	client pullreq.PullRequestClient,
	clusterClients []cluster.ClusterClient,
	flags map[string]string,
) error {
	err := client.UpdateStatus(
		ctx,
		"pending",
		whh.commandContext(commandApply),
		fmt.Sprintf(
			"Running for clusters %s",
			hashedClusterNames(clusterClients),
		),
		whh.settings.LogsURL,
	)
	if err != nil {
		log.Warnf("Error updating status: %+v", err)
	}

	var applyErr error

	statusOK := statusOKToApply(ctx, client)
	approved := client.Approved(ctx)
	behindBy := client.BehindBy()
	applyData := pullreq.ApplyCommentData{
		ClusterApplies:    []pullreq.ClusterApply{},
		PullRequestClient: client,
		Env:               whh.settings.Env,
	}

	if whh.settings.StrictCheck && !statusOK {
		applyErr = multilineError(
			"Cannot run apply because strict-check is set to true and commit status is not green.",
			"Please fix status and try again.",
		)
	} else if whh.settings.StrictCheck && !approved {
		applyErr = multilineError(
			"Cannot run apply because strict-check is set to true and request is not approved.",
			"Please get at least one approval and try again.",
		)
	} else if behindBy > 0 {
		applyErr = multilineError(
			fmt.Sprintf(
				"Cannot run apply because branch is behind %s by %d commits.",
				client.Base(),
				behindBy,
			),
			"Please re-merge and try again.",
		)
	} else {
		for _, clusterClient := range clusterClients {
			clusterName := clusterClient.Config().DescriptiveName()

			if err := clusterClient.Config().CheckVersion(whh.settings.Version); err != nil {
				applyErr = fmt.Errorf(
					"Failed version check for cluster %s: %+v",
					clusterName,
					err,
				)
				break
			}

			applyCtx, cancel := context.WithTimeout(ctx, applyTimeout)
			defer cancel()

			results, err := clusterClient.ApplyStructured(
				applyCtx,
				clusterClient.Config().AbsSubpath(),
				clusterClient.Config().ServerSideApply,
			)
			if err != nil {
				applyErr = fmt.Errorf("%+v", err)
				break
			}

			applyData.ClusterApplies = append(
				applyData.ClusterApplies,
				pullreq.ClusterApply{
					ClusterConfig: clusterClient.Config(),
					Results:       results,
				},
			)
		}
	}

	if applyErr != nil {
		client.PostErrorComment(ctx, whh.settings.Env, applyErr)
		client.UpdateStatus(
			ctx,
			"failure",
			whh.commandContext(commandApply),
			fmt.Sprintf(
				"Error running for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
		return applyErr
	}

	commentBody, err := pullreq.FormatApplyComment(applyData)
	if err != nil {
		client.UpdateStatus(
			ctx,
			"failure",
			whh.commandContext(commandApply),
			fmt.Sprintf(
				"Error running for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
		return err
	}

	err = client.PostComment(ctx, commentBody)
	if err != nil {
		log.Warnf("Error posting response: %+v", err)
	}

	noAutoMergeValue, ok := flags["no-auto-merge"]
	noAutoMerge := ok && (noAutoMergeValue == "" || strings.ToLower(noAutoMergeValue) == "true")

	if noAutoMerge {
		err = client.UpdateStatus(
			ctx,
			"failure",
			whh.commandContext(commandApply),
			fmt.Sprintf(
				"Successfully ran for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
	} else {
		err = client.UpdateStatus(
			ctx,
			"success",
			whh.commandContext(commandApply),
			fmt.Sprintf(
				"Successfully ran for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
	}

	if err != nil {
		log.Warnf("Error updating status: %+v", err)
	}

	return nil
}

func (whh *WebhookHandler) runDiffs(
	ctx context.Context,
	client pullreq.PullRequestClient,
	clusterClients []cluster.ClusterClient,
) error {
	err := client.UpdateStatus(
		ctx,
		"pending",
		whh.commandContext(commandDiff),
		fmt.Sprintf(
			"Running for clusters %s",
			hashedClusterNames(clusterClients),
		),
		whh.settings.LogsURL,
	)
	if err != nil {
		log.Warnf("Error updating status: %+v", err)
	}

	diffData := pullreq.DiffCommentData{
		ClusterDiffs:      []pullreq.ClusterDiff{},
		PullRequestClient: client,
		Env:               whh.settings.Env,
	}

	var diffErr error

	for _, clusterClient := range clusterClients {
		clusterName := clusterClient.Config().DescriptiveName()

		if err := clusterClient.Config().CheckVersion(whh.settings.Version); err != nil {
			diffErr = fmt.Errorf(
				"Failed version check for cluster %s: %+v",
				clusterName,
				err,
			)
			break
		}

		diffCtx, cancel := context.WithTimeout(ctx, diffTimeout)
		defer cancel()

		results, err := clusterClient.Diff(
			diffCtx,
			clusterClient.Config().AbsSubpath(),
			clusterClient.Config().ServerSideApply,
		)
		if err != nil {
			resultsStr := string(results)

			if resultsStr != "" {
				diffErr = fmt.Errorf(
					"%+v\nDiff output: %s",
					err,
					resultsStr,
				)
			} else {
				diffErr = fmt.Errorf("%+v", err)
			}
			break
		}

		diffData.ClusterDiffs = append(
			diffData.ClusterDiffs,
			pullreq.ClusterDiff{
				ClusterConfig: clusterClient.Config(),
				RawDiffs:      string(results),
			},
		)
	}

	if diffErr != nil {
		client.PostErrorComment(ctx, whh.settings.Env, diffErr)
		client.UpdateStatus(
			ctx,
			"failure",
			whh.commandContext(commandDiff),
			fmt.Sprintf(
				"Error running for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
		return diffErr
	}

	commentBody, err := pullreq.FormatDiffComment(diffData)
	if err != nil {
		client.PostErrorComment(ctx, whh.settings.Env, err)
		client.UpdateStatus(
			ctx,
			"failure",
			whh.commandContext(commandDiff),
			fmt.Sprintf(
				"Error running for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
		return err
	}

	err = client.PostComment(ctx, commentBody)
	if err != nil {
		log.Warnf("Error posting response: %+v", err)
	}

	err = client.UpdateStatus(
		ctx,
		"success",
		whh.commandContext(commandDiff),
		fmt.Sprintf(
			"Successfully ran for clusters %s",
			hashedClusterNames(clusterClients),
		),
		whh.settings.LogsURL,
	)
	if err != nil {
		log.Warnf("Error updating status: %+v", err)
	}

	return nil
}

func (whh *WebhookHandler) runStatus(
	ctx context.Context,
	client pullreq.PullRequestClient,
	clusterClients []cluster.ClusterClient,
) error {
	err := client.UpdateStatus(
		ctx,
		"pending",
		whh.commandContext(commandStatus),
		fmt.Sprintf(
			"Running for clusters %s",
			hashedClusterNames(clusterClients),
		),
		whh.settings.LogsURL,
	)
	if err != nil {
		log.Warnf("Error updating status: %+v", err)
	}

	statusData := pullreq.StatusCommentData{
		ClusterStatuses:   []pullreq.ClusterStatus{},
		PullRequestClient: client,
		Env:               whh.settings.Env,
	}

	var statusErr error

	for _, clusterClient := range clusterClients {
		statusCtx, cancel := context.WithTimeout(ctx, diffTimeout)
		defer cancel()

		results, err := clusterClient.Summary(statusCtx)
		if err != nil {
			resultsStr := string(results)

			if resultsStr != "" {
				statusErr = fmt.Errorf(
					"%+v\nStatus output: %s",
					err,
					resultsStr,
				)
			} else {
				statusErr = fmt.Errorf("%+v", err)
			}
			break
		}

		statusData.ClusterStatuses = append(
			statusData.ClusterStatuses,
			pullreq.ClusterStatus{
				ClusterConfig: clusterClient.Config(),
				HealthSummary: string(results),
			},
		)
	}

	if statusErr != nil {
		client.PostErrorComment(ctx, whh.settings.Env, statusErr)
		client.UpdateStatus(
			ctx,
			// Mark as success so that it doesn't block applying or merging
			"success",
			whh.commandContext(commandStatus),
			fmt.Sprintf(
				"Ran for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
		return statusErr
	}

	commentBody, err := pullreq.FormatStatusComment(statusData)
	if err != nil {
		client.UpdateStatus(
			ctx,
			// Mark as success so that it doesn't block applying or merging
			"success",
			whh.commandContext(commandStatus),
			fmt.Sprintf(
				"Ran for clusters %s",
				hashedClusterNames(clusterClients),
			),
			whh.settings.LogsURL,
		)
		return err
	}

	err = client.PostComment(ctx, commentBody)
	if err != nil {
		log.Warnf("Error posting response: %+v", err)
	}

	client.UpdateStatus(
		ctx,
		"success",
		whh.commandContext(commandStatus),
		fmt.Sprintf(
			"Successfully ran for clusters %s",
			hashedClusterNames(clusterClients),
		),
		whh.settings.LogsURL,
	)
	if err != nil {
		log.Warnf("Error updating status: %+v", err)
	}

	return nil
}

func (whh *WebhookHandler) runHelp(
	ctx context.Context,
	client pullreq.PullRequestClient,
	clusterClients []cluster.ClusterClient,
) error {
	helpData := pullreq.HelpCommentData{
		ClusterConfigs: []*config.ClusterConfig{},
		Env:            whh.settings.Env,
	}

	for _, clusterClient := range clusterClients {
		helpData.ClusterConfigs = append(
			helpData.ClusterConfigs,
			clusterClient.Config(),
		)
	}

	commentBody, err := pullreq.FormatHelpComment(helpData)
	if err != nil {
		return err
	}

	err = client.PostComment(ctx, commentBody)
	return err
}

func (whh *WebhookHandler) commandContext(cmd command) string {
	return fmt.Sprintf("kubeapply/%s (%s)", string(cmd), whh.settings.Env)
}

func (whh *WebhookHandler) incrementStat(
	name string,
	webhookContext *WebhookContext,
	commandStr string,
) error {
	tags := []string{
		fmt.Sprintf("owner:%s", webhookContext.owner),
		fmt.Sprintf("repo:%s", webhookContext.repo),
	}
	if commandStr != "" {
		tags = append(tags, fmt.Sprintf("command:%s", commandStr))
	}

	return whh.statsClient.Update(
		[]string{name},
		[]float64{1.0},
		tags,
		stats.StatTypeCount,
	)
}

func multilineError(lines ...string) error {
	return errors.New(strings.Join(lines, "\n"))
}

// hashedClusterNames returns a string of comma-separated, hashed cluster names for status
// descriptions. These are used to ensure that auto-merging isn't done until all clusters
// have been diffed and applied. We use hashes instead of the full names to ensure the list
// can fit in the description field.
func hashedClusterNames(clients []cluster.ClusterClient) string {
	names := []string{}
	for _, client := range clients {
		h := sha1.New()
		h.Write([]byte(client.Config().DescriptiveName()))
		hashStr := fmt.Sprintf("%x", h.Sum(nil))

		names = append(
			names,
			hashStr[0:8],
		)
	}
	return strings.Join(names, ",")
}
