package helm

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/Masterminds/semver/v3"
)

var helmVersionRegexp = regexp.MustCompile(`Version:"v([0-9a-zA-Z._-]+)"`)

// CheckHelmVersion checks that the helm version in the path matches
// the argument constraint.
func CheckHelmVersion(ctx context.Context, constraintStr string) error {
	helmVersion, err := getHelmVersion(ctx)
	if err != nil {
		return err
	}

	semVersion, err := semver.NewVersion(helmVersion)
	if err != nil {
		return err
	}

	constraint, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return err
	}

	if !constraint.Check(semVersion) {
		return fmt.Errorf(
			"version of helm in path (%s) does not satisfy constraint for kubeapply (%s). Please update helm and try again.",
			helmVersion,
			constraintStr,
		)
	}

	return nil
}

func getHelmVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "helm", "version")
	result, err := cmd.CombinedOutput()
	resultStr := string(result)

	if err != nil {
		return "", fmt.Errorf(
			"Error running 'helm version': %+v; output: %s",
			err,
			resultStr,
		)
	}

	matches := helmVersionRegexp.FindStringSubmatch(resultStr)
	if len(matches) != 2 {
		return "", fmt.Errorf(
			"Could not parse helm version from output: %s",
			resultStr,
		)
	}

	return matches[1], nil
}
