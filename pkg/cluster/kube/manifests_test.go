package kube

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	testManifest1 = `
# this is a comment

---
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: pod-log-reader
rules:
- apiGroups: [""]
  resources:
    - namespaces
    - pods
  verbs: ["get", "list", "watch"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluentbit
  namespace: monitoring
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::184402915685:role/ob-yolken.usw2.eks.fluentbit
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: pod-log-crb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: pod-log-reader
subjects:
- kind: ServiceAccount
  name: fluent-bit
  namespace: monitoring`

	testManifest2 = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: monitoring
  labels:
    app.kubernetes.io/name: fluentbit
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush           5
        Daemon          off
        Log_Level       info
        Parsers_File    parsers.conf
`
)

func TestGetManifests(t *testing.T) {
	outDir, err := ioutil.TempDir("", "data")
	if err != nil {
		assert.FailNow(t, "Cannot create tempDir: %+v", err)
	}
	defer os.RemoveAll(outDir)

	util.WriteFiles(
		t,
		outDir,
		map[string]string{
			"manifest1.yaml":     testManifest1,
			"dir/manifest2.yaml": testManifest2,
		},
	)

	manifests, err := GetManifests([]string{outDir})
	assert.Nil(t, err)
	assert.Equal(t, 4, len(manifests))

	SortManifests(manifests)

	kinds := []string{}

	for _, manifest := range manifests {
		kinds = append(kinds, manifest.Head.Kind)
	}

	assert.Equal(
		t,
		[]string{
			"ConfigMap",
			"ServiceAccount",
			"ClusterRole",
			"ClusterRoleBinding",
		},
		kinds,
	)

	assert.Equal(t, strings.TrimSpace(testManifest2), manifests[0].Contents)
}
