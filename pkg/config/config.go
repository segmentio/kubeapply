package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ghodss/yaml"
	"github.com/segmentio/kubeapply/pkg/util"
	log "github.com/sirupsen/logrus"
)

// ClusterConfig represents the configuration for a single Kubernetes cluster in a single
// region and environment / account.
type ClusterConfig struct {
	// Cluster is the name of the cluster.
	//
	// Required.
	Cluster string `json:"cluster"`

	// Region is the region for this cluster, e.g. us-west-2.
	//
	// Required.
	Region string `json:"region"`

	// Env is the environment or account for this cluster, e.g. production.
	//
	// Required.
	Env string `json:"env"`

	// UID is a unique identifier of this cluster. If set, kubeapply will validate that
	// uid of the cluster it is interacting with matches this value and abort otherwise.
	// This can help prevent against applying a configuration against the wrong cluster.
	//
	// You can fetch your cluster's UID by running:
	//
	//     kubectl get namespace kube-system -o json | jq -r .metadata.uid
	//
	// Optional.
	UID string `json:"uid"`

	// Charts is a URL for the default location of Helm charts.
	//
	// Required unless profile doesn't contain charts or all values files have explicit chart
	// URLs.
	Charts string `json:"charts"`

	// ProfilePath is the path to the profile directory for this cluster.
	//
	// Optional, defaults to "profile" if not set.
	ProfilePath string `json:"profilePath"`

	// ExpandedPath is the path to the results of expanding out all of the configs for this cluster.
	//
	// Optional, defaults to "expanded/[env]/[region]" if not set.
	ExpandedPath string `json:"expandedPath"`

	// Parameters are key/value pairs to be used for go templating.
	//
	// Optional.
	Parameters map[string]interface{} `json:"parameters"`

	// GithubIgnore indicates whether kubeapply-lambda webhooks should ignore this cluster.
	//
	// Optional, defaults to false.
	GithubIgnore bool `json:"ignore"`

	// VersionConstraint is a string version constraint against with the kubeapply binary
	// will be checked. See https://github.com/Masterminds/semver for details on the expected
	// format.
	//
	// Optional, defaults to no check.
	VersionConstraint string `json:"versionConstraint"`

	// KubeConfigPath is the path to a kubeconfig that can be used with this cluster.
	//
	// Optional, defaults to value set on command-line (when running kubeapply manually) or
	// automatically generated via AWS API (when running in lambdas case).
	KubeConfigPath string `json:"kubeConfig"`

	Subpath string `json:"-"`

	// For debugging / internal purposes only.
	fullPath        string
	relPath         string
	descriptiveName string
}

// LoadClusterConfig loads a config from a path on disk.
func LoadClusterConfig(path string, rootPath string) (*ClusterConfig, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &ClusterConfig{}
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	if config == nil || config.Cluster == "" {
		return nil, fmt.Errorf("File %s is not a valid cluster config", path)
	}

	if err := config.SetDefaults(path, rootPath); err != nil {
		return nil, err
	}

	return config, nil
}

// SetDefaults sets reasonable defaults for missing values in the current
// ClusterConfig.
func (c *ClusterConfig) SetDefaults(path string, rootPath string) error {
	var err error

	c.fullPath = path

	if rootPath == "" {
		c.relPath = path
	} else {
		c.relPath, err = filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
	}

	c.Subpath = "."

	if c.Env == "" {
		return errors.New("Env must be set")
	}
	if c.Region == "" {
		return errors.New("Region must be set")
	}

	c.descriptiveName = fmt.Sprintf(
		"%s:%s:%s",
		c.Env,
		c.Region,
		c.Cluster,
	)

	configDir := filepath.Dir(path)

	if c.ProfilePath == "" {
		c.ProfilePath = filepath.Join(configDir, "profile")
		log.Debugf(
			"ProfilePath not set explicitly, using default of %s",
			c.ProfilePath,
		)
	} else if !filepath.IsAbs(c.ProfilePath) {
		c.ProfilePath = filepath.Join(configDir, c.ProfilePath)
	}

	if c.ExpandedPath == "" {
		c.ExpandedPath = filepath.Join(
			configDir,
			"expanded",
			c.Env,
			c.Region,
		)
		log.Debugf(
			"ExpandedPath not set explicitly, using default of %s",
			c.ExpandedPath,
		)
	} else if !filepath.IsAbs(c.ExpandedPath) {
		c.ExpandedPath = filepath.Join(configDir, c.ExpandedPath)
	}

	ok, err := util.DirExists(c.ProfilePath)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("Profile path %s does not exist", c.ProfilePath)
	}

	return nil
}

// ShortRegion converts the region in the cluster config to a short form that
// may be used in some templates.
func (c ClusterConfig) ShortRegion() string {
	components := strings.Split(c.Region, "-")
	if len(components) != 3 {
		log.Warnf("Cannot convert region %s to short form", c.Region)
		return c.Region
	}

	return fmt.Sprintf("%s%s%s", components[0][0:2], components[1][0:1], components[2])
}

// CheckVersion checks that the version in the cluster config is compatible with this
// version of kubeapply.
func (c ClusterConfig) CheckVersion(version string) error {
	if c.VersionConstraint == "" {
		return nil
	}

	semVersion, err := semver.NewVersion(version)
	if err != nil {
		return err
	}

	constraint, err := semver.NewConstraint(c.VersionConstraint)
	if err != nil {
		return err
	}

	if !constraint.Check(semVersion) {
		return fmt.Errorf(
			"kubeapply version (%s) does not satisfy constraint in config (%s). If needed, update by running 'GO111MODULE=\"on\" go get github.com/segmentio/kubeapply/cmd/kubeapply'.",
			version,
			c.VersionConstraint,
		)
	}

	return nil
}

// AbsSubpath returns the absolute subpath of the expanded configs associated with
// this ClusterConfig.
func (c ClusterConfig) AbsSubpath() string {
	if c.Subpath != "" {
		return filepath.Join(c.ExpandedPath, c.Subpath)
	}
	return c.ExpandedPath
}

// DescriptiveName returns a descriptive name for this ClusterConfig.
func (c ClusterConfig) DescriptiveName() string {
	return c.descriptiveName
}

// FullPath returns the full path to this ClusterConfig.
func (c ClusterConfig) FullPath() string {
	return c.fullPath
}

// RelPath returns the relative path to this ClusterConfig.
func (c ClusterConfig) RelPath() string {
	return c.relPath
}

// PrettySubpath generates a Github-friendly format for the cluster subpath.
func (c ClusterConfig) PrettySubpath() string {
	if c.Subpath == "." {
		return "*all*"
	}
	return fmt.Sprintf("`%s`", c.Subpath)
}
