package apply

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubeJSONToObjectsList(t *testing.T) {
	objs, err := KubeJSONToObjects(loadFixtures(t, "testdata/objs_old.json", nil))
	require.Nil(t, err)

	assert.Equal(t, 3, len(objs))

	kinds := []string{}
	for _, obj := range objs {
		kinds = append(kinds, obj.Kind)
	}

	assert.Equal(t, []string{"Deployment", "ServiceAccount", "Service"}, kinds)
}

func TestKubeJSONToObjectsSingleObject(t *testing.T) {
	objs, err := KubeJSONToObjects(loadFixtures(t, "testdata/obj_old.json", nil))
	require.Nil(t, err)

	assert.Equal(t, 1, len(objs))

	kinds := []string{}
	for _, obj := range objs {
		kinds = append(kinds, obj.Kind)
	}

	assert.Equal(t, []string{"ServiceAccount"}, kinds)
}

func TestKubeJSONToObjectsWarningPrefix(t *testing.T) {
	objs, err := KubeJSONToObjects(
		loadFixtures(
			t,
			"testdata/obj_old.json",
			[]byte("WARN: This is a kubectl warning"),
		),
	)
	require.Nil(t, err)

	assert.Equal(t, 1, len(objs))

	kinds := []string{}
	for _, obj := range objs {
		kinds = append(kinds, obj.Kind)
	}

	assert.Equal(t, []string{"ServiceAccount"}, kinds)
}

func TestObjsToResults(t *testing.T) {
	oldObjs, err := KubeJSONToObjects(loadFixtures(t, "testdata/objs_old.json", nil))
	require.Nil(t, err)

	newObjs, err := KubeJSONToObjects(loadFixtures(t, "testdata/objs_new.json", nil))
	require.Nil(t, err)

	results, err := ObjsToResults(oldObjs, newObjs)
	require.Nil(t, err)

	// Convert all times to UTC so time comparisons work.
	for i := 0; i < len(results); i++ {
		results[i].CreatedAt = results[i].CreatedAt.UTC()
	}

	assert.Equal(
		t,
		[]Result{
			{
				Name:       "nginx-deployment",
				Namespace:  "default",
				Kind:       "Deployment",
				CreatedAt:  parseTime(t, "2020-06-18T04:38:46Z"),
				OldVersion: "58935",
				NewVersion: "58950",
				index:      0,
			},
			{
				Name:       "nginx-deployment",
				Namespace:  "default",
				Kind:       "ServiceAccount",
				CreatedAt:  parseTime(t, "2020-06-17T04:04:06Z"),
				OldVersion: "4817",
				NewVersion: "4820",
				index:      1,
			},
			{
				Name:       "nginx",
				Namespace:  "default",
				Kind:       "Service",
				CreatedAt:  parseTime(t, "2020-06-17T04:04:06Z"),
				OldVersion: "4818",
				NewVersion: "4818",
				index:      2,
			},
		},
		results,
	)
}

func loadFixtures(t *testing.T, path string, prefix []byte) []byte {
	contents, err := ioutil.ReadFile(path)
	require.Nil(t, err)

	if len(prefix) > 0 {
		contents = append(prefix, contents...)
	}

	return contents
}

func parseTime(t *testing.T, timeStr string) time.Time {
	result, err := time.Parse(time.RFC3339, timeStr)
	require.Nil(t, err)
	return result.UTC()
}
