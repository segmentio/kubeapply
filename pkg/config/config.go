package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/ghodss/yaml"
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

	// UID is a unique identifier of this cluster. Specifically, it is the unique
	// identifier of the kube-system namespace. If set, kubeapply will validate that
	// cluster it is interacting with has a matching kube-system namespace uid. This
	// can help prevent against accidentally running a kubeapply config on a similarly-
	// named cluster but in the wrong environment.
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

	// Profiles is a list of profiles for this cluster. Unlike the ProfilePath
	// above, these allow for multiple profiles in a single cluster. If these are set, then
	// ProfilePath will be ignored.
	//
	// Optional.
	Profiles []Profile `json:"profiles"`

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

	// ServerSideApply sets whether we should be using server-side applies and diffs for this
	// cluster.
	ServerSideApply bool `json:"serverSideApply"`

	// Subpath is the subset of the expanded configs that we want to diff or apply.
	Subpaths []string `json:"-"`

	// Profile is the current profile that's being used for config expansion.
	Profile *Profile `json:"-"`

	// For debugging / internal purposes only.
	fullPath        string
	relPath         string
	descriptiveName string
}

type Profile struct {
	// Name is the name of the profile.
	Name string `json:"name"`

	// URL is where the profile configs live.
	URL string `json:"url"`

	// Parameters are override parameters that will be merged on top of the global parameters
	// for this cluster.
	//
	// Optional.
	Parameters map[string]interface{} `json:"parameters"`
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

	c.Subpaths = []string{"."}

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

// AbsSubpaths returns the absolute subpaths of the expanded configs associated with
// this ClusterConfig.
func (c ClusterConfig) AbsSubpaths() []string {
	if len(c.Subpaths) > 0 {
		absSubpaths := []string{}

		for _, subPath := range c.Subpaths {
			absSubpaths = append(
				absSubpaths,
				filepath.Join(c.ExpandedPath, subPath),
			)
		}

		return absSubpaths
	}

	return []string{c.ExpandedPath}
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

// PrettySubpaths generates a Github-friendly format for the cluster subpaths.
func (c ClusterConfig) PrettySubpaths() string {
	subpathStrs := []string{}

	for s, subpath := range c.Subpaths {
		if s > 5 {
			subpathStrs = append(
				subpathStrs,
				"...",
			)
			break
		} else if subpath == "." {
			subpathStrs = append(subpathStrs, "*all*")
		} else {
			subpathStrs = append(subpathStrs, fmt.Sprintf("`%s`", subpath))
		}
	}

	return strings.Join(subpathStrs, ", ")
}

// PrettySubpathsList generates a Github-friendly, bulleted list for the cluster subpaths.
func (c ClusterConfig) PrettySubpathsList() string {
	subpathStrs := []string{}

	for s, subpath := range c.Subpaths {
		if s > 5 {
			subpathStrs = append(
				subpathStrs,
				fmt.Sprintf(
					"<li> ... %d others</li>",
					len(c.Subpaths)-5,
				),
			)
			break
		} else if subpath == "." {
			subpathStrs = append(subpathStrs, "<li>*all*</li>")
		} else {
			subpathStrs = append(subpathStrs, fmt.Sprintf("<li>`%s`</li>", subpath))
		}
	}

	return fmt.Sprintf(
		"<ul>%s</ul>",
		strings.Join(subpathStrs, ""),
	)
}

// SubpathCount generates the number of subpaths for Github comments.
func (c ClusterConfig) SubpathCount() int {
	return len(c.Subpaths)
}

// StarParams generates the base starlark params for this ClusterConfig.
func (c ClusterConfig) StarParams() map[string]interface{} {
	starParams := map[string]interface{}{
		"cluster":    c.Cluster,
		"env":        c.Env,
		"region":     c.Region,
		"parameters": c.Parameters,
	}
	for key, value := range c.Parameters {
		starParams[key] = value
	}

	return starParams
}
