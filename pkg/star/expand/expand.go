package expand

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	"github.com/segmentio/kubeapply/pkg/star/expand/skymod"
	log "github.com/sirupsen/logrus"
	"github.com/stripe/skycfg"
	"github.com/stripe/skycfg/gogocompat"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

// Adapated from example in https://github.com/stripe/skycfg/blob/master/_examples/k8s/main.go.
const k8sAPIPrefix = "k8s.io.api."

var k8sProtoMagic = []byte("k8s\x00")

// ExpandStar expands all starlark in the root directory, replacing each
// file with its YAML expansion.
func ExpandStar(
	expandRoot string,
	clusterRoot string,
	params map[string]interface{},
) error {
	starPaths := []string{}

	err := filepath.Walk(
		expandRoot,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(path, ".star") {
				return nil
			}

			starPaths = append(starPaths, path)

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// Skip over starlark files that don't have entrypoints
			if !strings.Contains(string(contents), "def main(ctx)") {
				return nil
			}

			log.Infof("Processing file %s", path)
			expandedStr, err := StarToYaml(path, clusterRoot, params)
			if err != nil {
				return err
			}

			// Replace suffix
			expandedPath := fmt.Sprintf("%s.yaml", path[0:len(path)-5])
			return ioutil.WriteFile(expandedPath, []byte(expandedStr), info.Mode())
		},
	)

	// Clean up .star files
	for _, starPath := range starPaths {
		os.Remove(starPath)
	}

	return err
}

// StarToYaml converts a starlark file into a YAML Kubernetes file.
func StarToYaml(path string, root string, params map[string]interface{}) (string, error) {
	objs, err := StarToObjs(path, root, params)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}

	for _, obj := range objs {
		fmt.Fprintf(buf, "---\n")

		var bytes []byte
		var err error

		if unknown, ok := obj.(*runtime.Unknown); ok {
			bytes = unknown.Raw
		} else {
			bytes, err = yaml.Marshal(obj)
			if err != nil {
				return "", err
			}
		}

		_, err = buf.Write(bytes)
		if err != nil {
			return "", err
		}
	}

	return string(buf.Bytes()), nil
}

// StarToObjs converts a starlark file into one or more Kubernetes
// objects.
func StarToObjs(
	path string,
	root string,
	params map[string]interface{},
) ([]runtime.Object, error) {
	reader, err := NewURLFileReader(root)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	config, err := skycfg.Load(
		context.Background(),
		path,
		skycfg.WithFileReader(reader),
		skycfg.WithProtoRegistry(gogocompat.ProtoRegistry()),
		skycfg.WithGlobals(
			map[string]starlark.Value{
				"util": skymod.UtilModule(),
				"yaml": skymod.YamlModule(),
			},
		),
	)
	if err != nil {
		return nil, err
	}

	skyParams := starlark.StringDict{}

	for key, val := range params {
		starVal, err := GoToStarValue(val)
		if err != nil {
			return nil, err
		}

		skyParams[key] = starVal
	}

	protos, err := config.Main(
		context.Background(),
		skycfg.WithVars(skyParams),
	)
	if err != nil {
		return nil, err
	}

	outputObjs := []runtime.Object{}

	for _, message := range protos {
		if obj, ok := message.(*runtime.Unknown); ok {
			outputObjs = append(outputObjs, obj)
			continue
		}

		gvk, err := gvkFromMsgType(message)
		if err != nil {
			return nil, err
		}

		bytes, err := marshal(message, *gvk)
		if err != nil {
			return nil, err
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(
			bytes,
			nil,
			nil,
		)

		outputObjs = append(outputObjs, obj)
	}

	return outputObjs, nil
}

// StarStrToObjs converts a starlark string into one or more Kubernetes
// objects. It's intended for testing.
func StarStrToObjs(
	starStr string,
	root string,
	params map[string]interface{},
) ([]runtime.Object, error) {
	tempDir, err := ioutil.TempDir("", "star")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	starPath := filepath.Join(tempDir, "file.star")

	err = ioutil.WriteFile(starPath, []byte(starStr), 0644)
	if err != nil {
		return nil, err
	}

	return StarToObjs(starPath, root, params)
}

// GoToStarValue converts the given go interface to the equivalent
// Starlark value.
func GoToStarValue(obj interface{}) (starlark.Value, error) {
	switch v := obj.(type) {
	case string:
		return starlark.String(v), nil
	case bool:
		return starlark.Bool(v), nil
	case int:
		return starlark.MakeInt(v), nil
	case int8:
		return starlark.MakeInt(int(v)), nil
	case int16:
		return starlark.MakeInt(int(v)), nil
	case int32:
		return starlark.MakeInt(int(v)), nil
	case int64:
		return starlark.MakeInt(int(v)), nil
	case uint:
		return starlark.MakeUint(uint(v)), nil
	case uint8:
		return starlark.MakeUint(uint(v)), nil
	case uint16:
		return starlark.MakeUint(uint(v)), nil
	case uint32:
		return starlark.MakeUint(uint(v)), nil
	case uint64:
		return starlark.MakeUint(uint(v)), nil

	// Starlark doesn't support floats; just convert them to ints
	case float32:
		return starlark.MakeInt(int(v)), nil
	case float64:
		return starlark.MakeInt(int(v)), nil

	default:
		rVal := reflect.ValueOf(obj)

		switch rVal.Kind() {
		case reflect.Ptr:
			return GoToStarValue(rVal.Elem().Interface())
		case reflect.Map:
			starMap := starlark.NewDict(rVal.Len())

			for _, key := range rVal.MapKeys() {
				val := rVal.MapIndex(key)

				starKey, err := GoToStarValue(key.Interface())
				if err != nil {
					return nil, err
				}
				starVal, err := GoToStarValue(val.Interface())
				if err != nil {
					return nil, err
				}

				err = starMap.SetKey(starKey, starVal)
			}

			return starMap, nil
		case reflect.Slice:
			elements := []starlark.Value{}

			for i := 0; i < rVal.Len(); i++ {
				element, err := GoToStarValue(rVal.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				elements = append(elements, element)
			}

			return starlark.NewList(elements), nil
		case reflect.Struct:
			entries := map[string]starlark.Value{}

			structType := rVal.Type()

			for i := 0; i < rVal.NumField(); i++ {
				if structType.Field(i).PkgPath != "" {
					// Field is unexported
					continue
				}

				starValue, err := GoToStarValue(rVal.Field(i).Interface())
				if err != nil {
					return nil, err
				}

				entries[structType.Field(i).Name] = starValue
			}

			return starlarkstruct.FromStringDict(starlarkstruct.Default, entries), nil
		default:
			return nil, fmt.Errorf("Unsupported type for value %+v: %s", obj, reflect.TypeOf(obj))
		}
	}
}

// gvkFromMsgType extracts the group, version, and kind from a Kubernetes
// proto struct.
func gvkFromMsgType(message proto.Message) (*schema.GroupVersionKind, error) {
	t := proto.MessageName(message)
	if !strings.HasPrefix(t, k8sAPIPrefix) {
		return nil, fmt.Errorf("unexpected message type: %+v", t)
	}

	ss := strings.Split(t[len(k8sAPIPrefix):], ".")

	// The API group doesn't always match the one implied by the proto name, unfortunately,
	// so we need some special-casing here.
	//
	// TODO: Improve this.
	if ss[0] == "core" {
		ss[0] = ""
	} else if ss[0] == "networking" {
		ss[0] = "extensions"
	} else if ss[0] == "rbac" {
		ss[0] = "rbac.authorization.k8s.io"
	}

	return &schema.GroupVersionKind{
		Group:   ss[0],
		Version: ss[1],
		Kind:    ss[2],
	}, nil
}

// marshal wraps a kubernetes proto message into a serialized Unknown
// struct that's understandable by the UniversalDeserializer.
func marshal(
	msg proto.Message,
	gvk schema.GroupVersionKind,
) ([]byte, error) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	v, k := gvk.ToAPIVersionAndKind()

	unknownBytes, err := proto.Marshal(
		&runtime.Unknown{
			TypeMeta: runtime.TypeMeta{
				APIVersion: v,
				Kind:       k,
			},
			Raw: msgBytes,
		},
	)
	if err != nil {
		return nil, err
	}
	return append(k8sProtoMagic, unknownBytes...), nil
}
