package convert

import (
	"testing"

	"github.com/segmentio/kubeapply/pkg/star/expand"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestYamlToStar(t *testing.T) {
	starStr, err := YamlToStar(
		[]string{
			"testdata/deployment.yaml",
			"testdata/statefulset.yaml",
		},
		Config{},
	)
	assert.Nil(t, err)

	objs, err := expand.StarStrToObjs(starStr, "", nil)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(objs))
	assert.IsType(t, &corev1.ConfigMap{}, objs[0])
	assert.IsType(t, &corev1.Service{}, objs[1])
	assert.IsType(t, &appsv1.Deployment{}, objs[2])
	assert.IsType(t, &appsv1.StatefulSet{}, objs[3])
}
