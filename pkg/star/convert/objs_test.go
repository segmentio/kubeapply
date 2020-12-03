package convert

import (
	"strings"
	"testing"

	"github.com/segmentio/kubeapply/pkg/star/expand"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type objsToStarTestCase struct {
	objs       []runtime.Object
	expStarStr string
	expErr     bool
}

func TestObjsToStar(t *testing.T) {
	testCases := []objsToStarTestCase{
		objsToStarTestCase{
			objs: []runtime.Object{},
			expStarStr: `
def main(ctx):

  return [
  ]`,
		},
		objsToStarTestCase{
			objs: []runtime.Object{
				&corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-config",
					},
					Data: map[string]string{
						"my_file.txt": "Line1\nLine2\nLine3",
					},
				},
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  "main",
								Image: "my_image:latest",
								Ports: []corev1.ContainerPort{
									corev1.ContainerPort{
										ContainerPort: 80,
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										// Don't have multiple items here to avoid
										// test flakiness due to ordering
										corev1.ResourceCPU: resource.MustParse("300m"),
									},
									Limits: corev1.ResourceList{
										// Don't have multiple items here to avoid
										// test flakiness due to ordering
										corev1.ResourceMemory: resource.MustParse("5G"),
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							corev1.Volume{
								Name: "test-volume",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/vol/path",
									},
								},
							},
						},
					},
				},
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
			expStarStr: `
corev1 = proto.package("k8s.io.api.core.v1")
metav1 = proto.package("k8s.io.apimachinery.pkg.apis.meta.v1")

def main(ctx):
  test_config_configmap = corev1.ConfigMap(
    metadata = metav1.ObjectMeta(
      name = "test-config",
    ),
    data = {
      "my_file.txt": """Line1
Line2
Line3""",
    },
  )
  pod = corev1.Pod(
    spec = corev1.PodSpec(
      volumes = [
        corev1.Volume(
          name = "test-volume",
          volumeSource = corev1.VolumeSource(
            hostPath = corev1.HostPathVolumeSource(
              path = "/vol/path",
            ),
          ),
        ),
      ],
      containers = [
        corev1.Container(
          name = "main",
          image = "my_image:latest",
          ports = [
            corev1.ContainerPort(
              containerPort = 80,
            ),
          ],
          resources = corev1.ResourceRequirements(
            limits = {
              "memory": util.quantity("5G"),
            },
            requests = {
              "cpu": util.quantity("300m"),
            },
          ),
        ),
      ],
    ),
  )
  pod1 = corev1.Pod(
  )

  return [
    test_config_configmap,
    pod,
    pod1,
  ]`,
		},
	}

	for _, testCase := range testCases {
		result, err := ObjsToStar(testCase.objs)
		if testCase.expErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		assert.Equal(
			t,
			strings.TrimSpace(testCase.expStarStr),
			strings.TrimSpace(result),
		)

		// Re-evaluate the generated starlark to make sure we get back
		// to the original Kubernetes objects
		starObjs, err := expand.StarStrToObjs(result, "", nil)
		assert.Nil(t, err)

		if len(testCase.objs) != len(starObjs) {
			assert.FailNowf(
				t,
				"Wrong length of returned objects from starlark evaluation",
				"Expected %d, got %d", len(testCase.objs), len(starObjs),
			)
		}

		for o, expObj := range testCase.objs {
			util.CompareJSONObjs(t, expObj, starObjs[o])
		}
	}
}
