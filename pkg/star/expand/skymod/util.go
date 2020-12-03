package skymod

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/stripe/skycfg"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubectl/pkg/scheme"
)

var splitRegexp = regexp.MustCompile("(^|\n)\\s*---\\s*\n")

// UtilModule defines a skycfg module with various helper functions
// in it.
func UtilModule() starlark.Value {
	return &Module{
		Name: "util",
		Attrs: starlark.StringDict{
			"quantity": quantity(),
			"intOrStr": intOrStr(),
			"fromYaml": fromYaml(),
			"rawYaml":  rawYaml(),
		},
	}
}

func quantity() starlark.Callable {
	return starlark.NewBuiltin("util.quantity", fnQuantity)
}

func fnQuantity(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var val starlark.String
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &val); err != nil {
		return nil, err
	}

	valStr, ok := starlark.AsString(val)
	if !ok {
		return nil, fmt.Errorf("Could not get string from quantity argument")
	}

	quantity, err := resource.ParseQuantity(valStr)
	if err != nil {
		return nil, err
	}

	return skycfg.NewProtoMessage(&quantity), nil
}

func intOrStr() starlark.Callable {
	return starlark.NewBuiltin("util.intOrStr", fnIntOrStr)
}

func fnIntOrStr(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var v starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
		return nil, err
	}

	var intOrStr intstr.IntOrString

	switch val := v.(type) {
	case starlark.String:
		unquotedStr, err := strconv.Unquote(val.String())
		if err != nil {
			return nil, err
		}

		intOrStr = intstr.FromString(unquotedStr)
	case starlark.Int:
		intVal, _ := val.Int64()
		intOrStr = intstr.FromInt(int(intVal))
	default:
		return nil, fmt.Errorf("Argument to IntOrStr must be int or string")
	}

	return skycfg.NewProtoMessage(&intOrStr), nil
}

func fromYaml() starlark.Callable {
	return starlark.NewBuiltin("util.fromYaml", fnFromYaml)
}

func fnFromYaml(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var val starlark.String
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &val); err != nil {
		return nil, err
	}

	valStr, ok := starlark.AsString(val)
	if !ok {
		return nil, fmt.Errorf("Could not get string from fromYaml argument")
	}

	objs, err := YamlStrToObjs(valStr)
	if err != nil {
		return nil, err
	}

	starObjs := []starlark.Value{}

	for _, obj := range objs {
		objProto, ok := obj.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("Object %+v is not a valid proto", obj)
		}

		starObjs = append(starObjs, skycfg.NewProtoMessage(objProto))
	}

	return starlark.NewList(starObjs), nil
}

func rawYaml() starlark.Callable {
	return starlark.NewBuiltin("util.rawYaml", fnRawYaml)
}

func fnRawYaml(
	t *starlark.Thread,
	fn *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var val starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &val); err != nil {
		return nil, err
	}

	var starDict *starlark.Dict

	switch v := val.(type) {
	case *starlark.Dict:
		starDict = v
	case *starlarkstruct.Struct:
		d := starlark.StringDict{}
		v.ToStringDict(d)
		starDict = starlark.NewDict(10)

		for _, key := range d.Keys() {
			starDict.SetKey(starlark.String(key), d[key])
		}
	default:
		return nil, fmt.Errorf("Argument to rawYaml must be either dict or struct")
	}

	yamlStar, err := starlark.Call(
		t,
		yamlMarshal(),
		starlark.Tuple{
			starDict,
		},
		[]starlark.Tuple{},
	)
	if err != nil {
		return nil, err
	}

	yamlStr, ok := starlark.AsString(yamlStar)
	if !ok {
		return nil, fmt.Errorf("Cannot convert yamlMarshal result to string")
	}

	wrappedMsg := &runtime.Unknown{
		Raw: []byte(yamlStr),
	}

	return skycfg.NewProtoMessage(wrappedMsg), nil
}

// YamlStrToObjs converts a YAML string into a slice of Kubernetes
// objects.
func YamlStrToObjs(yamlStr string) ([]runtime.Object, error) {
	yamlDocs := splitRegexp.Split(yamlStr, -1)
	objs := []runtime.Object{}

	for _, yamlDoc := range yamlDocs {
		trimmedDoc := strings.TrimSpace(yamlDoc)

		if trimmedDoc == "" {
			continue
		}

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(
			[]byte(trimmedDoc),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	return objs, nil
}
