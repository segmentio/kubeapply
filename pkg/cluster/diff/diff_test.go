package diff

import (
	"testing"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffKube(t *testing.T) {
	results, err := DiffKube("testdata/old", "testdata/new", false)
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
@@ -1,33 +1,32 @@
 apiVersion: apps/v1
 kind: Deployment
 metadata:
-  generation: 1
+  generation: 2
   annotations:
-    k2.segment.com/context: "old"
+    k2.segment.com/context: "new"
   labels:
     app: echoserver
-    app.kubernetes.io/instance: argocd-label
     k2.segment.com/app: echoserver
     k2.segment.com/argo-cd: echoserver
     k2.segment.com/build: master-008c1d0
-    k2.segment.com/deploy: cd123456
-    k2.segment.com/repo: test
-    enabled: false
+    k2.segment.com/deploy: 7c123456
+    k2.segment.com/repo: kubeapply_test
+    enabled: true
   name: echoserver
   namespace: apps
 spec:
-  replicas: 1
+  replicas: 3
   selector:
     matchLabels:
       app: echoserver
   template:
     metadata:
       annotations:
-        k2.segment.com/context: "old"
+        k2.segment.com/context: "new"
       labels:
         app: echoserver
-        enabled: false
-        version: "1.0"
+        enabled: true
+        version: "1.2"
         value: "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789... (71 chars omitted)
     spec:
       containers:
`,
		results[0].RawDiff,
	)
	assert.Equal(
		t,
		9,
		results[0].NumAdded,
	)
	assert.Equal(
		t,
		10,
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

func TestDiffKubeWithShortDiff(t *testing.T) {
	results, err := DiffKube("testdata/old", "testdata/new", true)
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
@@ -3,11 +3,11 @@
 metadata:
   labels:
     app: echoserver
-    enabled: false
+    enabled: true
   name: echoserver
   namespace: apps
 spec:
-  replicas: 1
+  replicas: 3
   selector:
     matchLabels:
       app: echoserver
@@ -15,7 +15,7 @@
     metadata:
       labels:
         app: echoserver
-        enabled: false
+        enabled: true
         value: "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789... (71 chars omitted)
     spec:
       containers:
`,
		results[0].RawDiff,
	)
	assert.Equal(
		t,
		3,
		results[0].NumAdded,
	)
	assert.Equal(
		t,
		3,
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
