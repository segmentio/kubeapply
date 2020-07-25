package kube

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
)

// KindOrder specifies the order in which Kubernetes resource types should be applied. Adapted from
// the list in https://github.com/helm/helm/blob/master/pkg/releaseutil/kind_sorter.go.
var KindOrder []string = []string{
	"Namespace",
	"NetworkPolicy",
	"ResourceQuota",
	"LimitRange",
	"PodSecurityPolicy",
	"PodDisruptionBudget",
	"Secret",
	"ConfigMap",
	"ConfigMapList",
	"StorageClass",
	"PersistentVolume",
	"PersistentVolumeClaim",
	"ServiceAccount",
	"CustomResourceDefinition",
	"ClusterRole",
	"ClusterRoleList",
	"ClusterRoleBinding",
	"ClusterRoleBindingList",
	"Role",
	"RoleList",
	"RoleBinding",
	"RoleBindingList",
	"Service",
	"DaemonSet",
	"Pod",
	"ReplicationController",
	"ReplicaSet",
	"Deployment",
	"HorizontalPodAutoscaler",
	"StatefulSet",
	"Job",
	"CronJob",
	"Ingress",
	"APIService",
}

var KindithoutMetadata []string = []string{
	"ConfigMapList",
	"RoleBindingList",
	"RoleList",
}

type Manifest struct {
	Path     string
	Head     SimpleHeader
	Contents string
}

// SimpleHeader is a simplified header used to getting basic metadata from
// Kubernetes YAML files. Adapted from same struct in Helm repo.
type SimpleHeader struct {
	Version  string `json:"apiVersion"`
	Kind     string `json:"kind,omitempty"`
	Metadata *struct {
		Name        string            `json:"name"`
		Namespace   string            `json:"namespace"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata,omitempty"`
}

// GetManifests recursively parses all of the manifests in the argument path.
func GetManifests(path string) ([]Manifest, error) {
	results := []Manifest{}

	err := filepath.Walk(
		path,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(subPath, ".yaml") {
				return nil
			}

			contents, err := ioutil.ReadFile(subPath)
			if err != nil {
				return err
			}

			trimmedFile := strings.TrimSpace(string(contents))
			manifestStrs := sep.Split(trimmedFile, -1)

			for _, manifestStr := range manifestStrs {
				manifestStr = strings.TrimSpace(manifestStr)
				if isEmpty(manifestStr) {
					continue
				}

				head := SimpleHeader{}
				err := yaml.Unmarshal([]byte(manifestStr), &head)
				if err != nil {
					log.Warnf("Could not parse head from %s; skipping file", subPath)
					continue
				}

				if head.Metadata == nil && !contains(KindithoutMetadata, head.Kind) {
					log.Warnf(
						"Could not read metadata from manifest %s in file %s",
						manifestStr,
						subPath,
					)
					continue
				}

				results = append(
					results,
					Manifest{
						Path:     subPath,
						Contents: manifestStr,
						Head:     head,
					},
				)
			}

			return nil
		},
	)

	return results, err
}

func contains(list []string, str string) bool {
	for _, v := range list {
		if str == v {
			return true
		}
	}
	return false
}

func isEmpty(contents string) bool {
	lines := strings.Split(contents, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			return false
		}
	}

	return true
}

// SortManifests sorts the provided manifest slice using the KindOrder above.
// Ties within the same type are broken by (namespace, name).
func SortManifests(manifests []Manifest) {
	orderMap := map[string]int{}

	for k, kind := range KindOrder {
		orderMap[kind] = k
	}

	sort.Slice(
		manifests,
		func(i, j int) bool {
			manifest1 := manifests[i]
			manifest2 := manifests[j]

			var kindOrder1, kindOrder2 int
			var namespace1, namespace2 string
			var name1, name2 string

			if manifest1.Head.Kind != "" {
				if order, ok := orderMap[manifest1.Head.Kind]; ok {
					kindOrder1 = order
				} else {
					kindOrder1 = len(orderMap)
				}

				if manifest1.Head.Metadata != nil {
					name1 = manifest1.Head.Metadata.Name
					namespace1 = manifest1.Head.Metadata.Namespace
				}
			} else {
				kindOrder1 = len(orderMap)
			}

			if manifest2.Head.Kind != "" {
				if order, ok := orderMap[manifest2.Head.Kind]; ok {
					kindOrder2 = order
				} else {
					kindOrder2 = len(orderMap)
				}

				if manifest2.Head.Metadata != nil {
					name2 = manifest2.Head.Metadata.Name
					namespace2 = manifest2.Head.Metadata.Namespace
				}
			} else {
				kindOrder2 = len(orderMap)
			}

			if kindOrder1 < kindOrder2 {
				return true
			} else if kindOrder1 == kindOrder2 && namespace1 < namespace2 {
				return true
			} else if kindOrder1 == kindOrder2 && namespace1 == namespace2 && name1 < name2 {
				return true
			}

			return false
		},
	)
}
