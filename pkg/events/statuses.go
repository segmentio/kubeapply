package events

import (
	"context"
	"regexp"
	"strings"

	"github.com/segmentio/kubeapply/pkg/pullreq"
	log "github.com/sirupsen/logrus"
)

var (
	contextRegexp     = regexp.MustCompile("kubeapply/(\\S+) [(](\\S+)[)]")
	descriptionRegexp = regexp.MustCompile("for clusters (\\S+)")
)

func statusAllGreen(
	ctx context.Context,
	client pullreq.PullRequestClient,
) bool {
	statuses, err := client.Statuses(ctx)
	if err != nil {
		log.Warnf("Error getting pull request statuses: %+v; assuming not ok", err)
		return false
	}

	for _, status := range statuses {
		if !status.IsSuccess() {
			log.Infof("Status is not green: %+v", status)
			return false
		}
	}

	log.Info("All statuses green")
	return true
}

func statusOKToApply(
	ctx context.Context,
	client pullreq.PullRequestClient,
) bool {
	statuses, err := client.Statuses(ctx)
	if err != nil {
		log.Warnf("Error getting pull request statuses: %+v; assuming not ok", err)
		return false
	}

	for _, status := range statuses {
		if !strings.Contains(status.Context, "kubeapply/apply") && !status.IsSuccess() {
			log.Infof("Non-apply status is not green: %+v", status)
			return false
		}
	}

	log.Info("All non-apply statuses are green")
	return true
}

func statusWorkflowCompleted(
	ctx context.Context,
	client pullreq.PullRequestClient,
) bool {
	statuses, err := client.Statuses(ctx)
	if err != nil {
		log.Warnf("Error getting pull request statuses: %+v; assuming not ok", err)
		return false
	}

	allClusters := map[string]struct{}{}
	diffedClusters := map[string]struct{}{}
	appliedClusters := map[string]struct{}{}

	for _, status := range statuses {
		contextMatches := contextRegexp.FindStringSubmatch(status.Context)
		if len(contextMatches) != 3 {
			continue
		}

		command := contextMatches[1]

		descriptionMatches := descriptionRegexp.FindStringSubmatch(
			status.Description,
		)
		if len(descriptionMatches) != 2 {
			continue
		}

		clusterNames := strings.Split(descriptionMatches[1], ",")

		for _, clusterName := range clusterNames {
			allClusters[clusterName] = struct{}{}

			if command == "diff" {
				// Consider all diffs
				diffedClusters[clusterName] = struct{}{}
			} else if command == "apply" && status.IsSuccess() {
				// Only consider applies that are successful
				appliedClusters[clusterName] = struct{}{}
			}
		}
	}

	if len(allClusters) == 0 {
		log.Info("No command statuses found, workflow not complete")
		return false
	}

	for clusterName := range allClusters {
		_, diffed := diffedClusters[clusterName]
		_, applied := appliedClusters[clusterName]
		if !diffed || !applied {
			log.Warnf(
				"Cluster %s is not fully diffed and applied: %v, %v",
				clusterName,
				diffed,
				applied,
			)
			return false
		}
	}

	log.Info("All clusters have been diffed and applied")
	return true
}
