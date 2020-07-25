package events

import (
	"context"
	"regexp"
	"strings"

	"github.com/segmentio/kubeapply/pkg/pullreq"
	log "github.com/sirupsen/logrus"
)

var contextRegexp = regexp.MustCompile("kubeapply/(\\S+) [(](\\S+)[)]")

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

	allEnvs := map[string]struct{}{}
	diffedEnvs := map[string]struct{}{}
	appliedEnvs := map[string]struct{}{}

	for _, status := range statuses {
		matches := contextRegexp.FindStringSubmatch(status.Context)
		if len(matches) != 3 {
			continue
		}

		command := matches[1]
		env := matches[2]

		allEnvs[env] = struct{}{}

		if command == "diff" {
			// Consider all diffs
			diffedEnvs[env] = struct{}{}
		} else if command == "apply" && status.IsSuccess() {
			// Only consider applies that are successful
			appliedEnvs[env] = struct{}{}
		}
	}

	if len(allEnvs) == 0 {
		log.Info("No command statuses found, workflow not complete")
		return false
	}

	for env := range allEnvs {
		_, diffed := diffedEnvs[env]
		_, applied := appliedEnvs[env]
		if !diffed || !applied {
			log.Warnf(
				"Env %s is not fully diffed and applied: %v, %v",
				env,
				diffed,
				applied,
			)
			return false
		}
	}

	log.Info("All clusters have been diffed and applied")
	return true
}
