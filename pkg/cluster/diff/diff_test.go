package diff

import (
	"testing"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffKube(t *testing.T) {
	results, err := DiffKube("testdata/old", "testdata/new")
	require.NoError(t, err)
	require.Equal(t, 3, len(results))

	names := []string{}
	for _, result := range results {
		names = append(names, result.Name)
	}

	assert.Equal(
		t,
		[]string{
			"file1.yaml",
			"file2.yaml",
			"file3.yaml",
		},
		names,
	)
	assert.Equal(
		t,
		`--- Server:file1.yaml
+++ Local:file1.yaml
@@ -4,7 +4,7 @@
   name: echoserver
   namespace: apps
 spec:
-  replicas: 1
+  replicas: 3
   selector:
     matchLabels:
       app: echoserver
@@ -12,7 +12,7 @@
     metadata:
       labels:
         app: echoserver
-        version: "1.0"
+        version: "1.2"
         value: "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789... (71 chars omitted)
     spec:
       containers:
`,
		results[0].RawDiff,
	)
	assert.Equal(
		t,
		2,
		results[0].NumAdded,
	)
	assert.Equal(
		t,
		2,
		results[0].NumRemoved,
	)
	assert.Equal(
		t,
		&apply.TypedKubeObj{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			KubeMetadata: apply.KubeMetadata{
				Name:      "echoserver",
				Namespace: "apps",
			},
		},
		results[0].Object,
	)
}
