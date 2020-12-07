package skymod

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stripe/skycfg"
	"github.com/stripe/skycfg/gogocompat"
	"go.starlark.net/starlark"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFromYaml(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "star")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	starPath := filepath.Join(tempDir, "main.star")
	err = ioutil.WriteFile(
		starPath,
		[]byte(`
def main(ctx):
  return util.fromYaml("""
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: special-config
  namespace: default
---
apiVersion: v1
kind: Service
metadata:
  name: kafka
  namespace: centrifuge
  labels:
    app: kafka
  """)`,
		),
		0644,
	)
	assert.Nil(t, err)

	config, err := skycfg.Load(
		context.Background(),
		starPath,
		skycfg.WithProtoRegistry(gogocompat.ProtoRegistry()),
		skycfg.WithGlobals(
			map[string]starlark.Value{
				"util": UtilModule(),
			},
		),
	)
	assert.Nil(t, err)

	protos, err := config.Main(
		context.Background(),
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(protos))
}

func TestRawYaml(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "star")
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	starPath := filepath.Join(tempDir, "main.star")
	err = ioutil.WriteFile(
		starPath,
		[]byte(`
def main(ctx):
  return [
    util.rawYaml(
      {
        'key1': 'value1',
        'key2': 'value2',
      },
    ),
    util.rawYaml(
      struct(
        key3 = 'value3',
        key4 = 'value4',
      ),
    ),
  ]`,
		),
		0644,
	)

	config, err := skycfg.Load(
		context.Background(),
		starPath,
		skycfg.WithProtoRegistry(gogocompat.ProtoRegistry()),
		skycfg.WithGlobals(
			map[string]starlark.Value{
				"util": UtilModule(),
			},
		),
	)
	assert.Nil(t, err)

	protos, err := config.Main(
		context.Background(),
	)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(protos))
	assert.IsType(t, &runtime.Unknown{}, protos[0])
	assert.IsType(t, &runtime.Unknown{}, protos[1])

	assert.Equal(
		t,
		"key1: value1\nkey2: value2\n",
		string(protos[0].(*runtime.Unknown).Raw),
	)
	assert.Equal(
		t,
		"key3: value3\nkey4: value4\n",
		string(protos[1].(*runtime.Unknown).Raw),
	)
}
