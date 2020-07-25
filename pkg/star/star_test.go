package star

import (
	"sort"
	"testing"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type starToObjsTestCase struct {
	starStr string
	params  map[string]interface{}
	expErr  bool
	expObjs []runtime.Object
}

func TestStarToObjs(t *testing.T) {
	testCases := []starToObjsTestCase{
		{
			starStr: "xxx",
			expErr:  true,
		},
		{
			starStr: `
def main(ctx):
  return [util.rawYaml({'key1': 'value1'})]`,
			expObjs: []runtime.Object{
				&runtime.Unknown{
					Raw: []byte("key1: value1\n"),
				},
			},
		},
		{
			starStr: `
corev1 = proto.package("k8s.io.api.core.v1")
metav1 = proto.package("k8s.io.apimachinery.pkg.apis.meta.v1")

def main(ctx):
  return [
    corev1.Pod(
      spec = corev1.PodSpec(
        containers = [
          corev1.Container(
            name = ctx.vars["name"],
            image = "my_image" + ":latest",
            ports = [
              corev1.ContainerPort(containerPort = 80)
            ],
            resources = corev1.ResourceRequirements(
              requests = {
                "cpu": util.quantity("300m"),
                "memory": util.quantity("2G"),
              },
              limits = {
                "cpu": util.quantity("300m"),
                "memory": util.quantity("5G"),
              },
            ),
          ),
        ],
      ),
    ),
    corev1.Service(
      metadata = metav1.ObjectMeta(
        name = "kafka",
        namespace = "centrifuge",
        labels = {
          "app": "kafka",
        },
      ),
      spec = corev1.ServiceSpec(
        ports = [
          corev1.ServicePort(
            name = "broker",
            port = 9092,
            targetPort = util.intOrStr("44445"),
          ),
        ],
        selector = {
          "app": "kafka",
        },
      ),
    ),
  ]`,
			params: map[string]interface{}{
				"name": "main",
			},
			expObjs: []runtime.Object{
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "main",
								Image: "my_image:latest",
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 80,
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("300m"),
										corev1.ResourceMemory: resource.MustParse("2G"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("300m"),
										corev1.ResourceMemory: resource.MustParse("5G"),
									},
								},
							},
						},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Service",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kafka",
						Namespace: "centrifuge",
						Labels: map[string]string{
							"app": "kafka",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name: "broker",
								Port: 9092,
								TargetPort: intstr.IntOrString{
									Type:   1,
									IntVal: 0,
									StrVal: "44445",
								},
							},
						},
						Selector: map[string]string{
							"app": "kafka",
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		objs, err := StarStrToObjs(testCase.starStr, "root", testCase.params)
		if testCase.expErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		if len(testCase.expObjs) != len(objs) {
			assert.FailNowf(
				t,
				"Wrong length of returned objects",
				"Expected %d, got %d", len(testCase.expObjs), len(objs),
			)
		}

		for o, expObj := range testCase.expObjs {
			if rawObj, ok := expObj.(*runtime.Unknown); ok {
				// Unknown objects can't be serialized to JSON, so just
				// compare them directly
				assert.Equal(t, rawObj, objs[o])
			} else {
				// Compare JSON to make debugging easier
				util.CompareJsonObjs(t, expObj, objs[o])
			}
		}
	}
}

func TestStarToObjsEndToEnd(t *testing.T) {
	objs, err := StarToObjs("./testdata/app.star", "testdata", nil)
	assert.Nil(t, err)

	assert.Equal(t, 3, len(objs))

	assert.IsType(t, &appsv1.Deployment{}, objs[0])
	assert.IsType(t, &appsv1.Deployment{}, objs[1])
	assert.IsType(t, &corev1.Service{}, objs[2])
}

type goToStarValueTestCase struct {
	goVal      interface{}
	expErr     bool
	expStarVal interface{}
}

func TestGoToStarValue(t *testing.T) {
	testCases := []goToStarValueTestCase{
		{
			goVal:      "hello",
			expStarVal: starlark.String("hello"),
		},
		{
			goVal:      true,
			expStarVal: starlark.Bool(true),
		},
		{
			goVal:      4123,
			expStarVal: starlark.MakeInt(4123),
		},
		{
			goVal: func() *int {
				i := 4123
				return &i
			}(),
			expStarVal: starlark.MakeInt(4123),
		},
		{
			goVal: []interface{}{"elem1", "elem2", 1234},
			expStarVal: starlark.NewList(
				[]starlark.Value{
					starlark.String("elem1"),
					starlark.String("elem2"),
					starlark.MakeInt(1234),
				},
			),
		},
		{
			goVal: map[string]interface{}{
				"key1": "value1",
			},
			expStarVal: func() *starlark.Dict {
				d := starlark.NewDict(1)
				d.SetKey(starlark.String("key1"), starlark.String("value1"))
				return d
			}(),
		},
		{
			goVal: struct {
				Name        string
				Value       int
				privateName string
			}{
				Name:        "testName",
				Value:       1234,
				privateName: "private",
			},
			expStarVal: starlarkstruct.FromStringDict(
				starlarkstruct.Default,
				map[string]starlark.Value{
					"Name":  starlark.String("testName"),
					"Value": starlark.MakeInt(1234),
				},
			),
		},
		{
			goVal:  func() {},
			expErr: true,
		},
	}

	for _, testCase := range testCases {
		result, err := GoToStarValue(testCase.goVal)

		if testCase.expErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		assert.Equal(t, testCase.expStarVal, result)
	}
}

func TestGoToStarValueMap(t *testing.T) {
	result, err := GoToStarValue(
		map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": 1234,
			"key4": []string{"value4", "value5"},
		},
	)
	assert.Nil(t, err)
	assert.IsType(t, &starlark.Dict{}, result)

	items := result.(*starlark.Dict).Items()

	sort.Slice(
		items,
		func(a, b int) bool {
			if len(items[a]) == 0 || len(items[b]) == 0 {
				return true
			}

			return items[a][0].String() < items[b][0].String()
		},
	)

	assert.Equal(
		t, []starlark.Tuple{
			[]starlark.Value{
				starlark.String("key1"),
				starlark.String("value1"),
			},
			[]starlark.Value{
				starlark.String("key2"),
				starlark.String("value2"),
			},
			[]starlark.Value{
				starlark.String("key3"),
				starlark.MakeInt(1234),
			},
			[]starlark.Value{
				starlark.String("key4"),
				starlark.NewList(
					[]starlark.Value{
						starlark.String("value4"),
						starlark.String("value5"),
					},
				),
			},
		},
		items,
	)
}
