package convert

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ObjsToStar converts a slice of Kubernetes objects into a starlark string.
func ObjsToStar(objs []runtime.Object, config Config) (string, error) {
	usedModules := map[string]struct{}{}
	objsDefs := []string{}
	objNames := []string{}
	objNamesMap := map[string]struct{}{}

	for _, obj := range objs {
		objProto, ok := obj.(proto.Message)

		if !ok {
			return "", fmt.Errorf("Could not convert message to proto message")
		}

		name := objName(obj, objNamesMap)
		objNames = append(objNames, name)
		objNamesMap[name] = struct{}{}

		out := &bytes.Buffer{}
		err := walkObj(reflect.ValueOf(objProto), 0, usedModules, config, out)
		if err != nil {
			return "", err
		}
		objsDefs = append(objsDefs, string(out.Bytes()))
	}

	moduleNames := []string{}

	for module := range usedModules {
		moduleNames = append(moduleNames, module)
	}

	sort.Slice(moduleNames, func(a, b int) bool {
		return moduleNames[a] < moduleNames[b]
	})

	resultLines := []string{}

	for _, module := range moduleNames {
		importName, err := ModuleToImportName(module)
		if err != nil {
			return "", err
		}

		resultLines = append(
			resultLines,
			fmt.Sprintf(
				"%s = proto.package(\"%s\")",
				module,
				importName,
			),
		)
	}

	resultLines = append(resultLines, "")

	if config.Entrypoint == "" || config.Entrypoint == "main" {
		resultLines = append(resultLines, "def main(ctx):")
	} else {
		resultLines = append(resultLines, fmt.Sprintf("def %s(", config.Entrypoint))
		resultLines = append(resultLines, fmt.Sprintf("%sctx,", indentToLevel(1)))
		for _, arg := range config.Args {
			resultLines = append(
				resultLines,
				fmt.Sprintf(
					"%s%s = %s,",
					indentToLevel(1),
					arg.Name,
					arg.DefaultValueStr(),
				),
			)
		}
		resultLines = append(resultLines, "):")

		requiredCount := 0

		for _, arg := range config.Args {
			if arg.Required {
				resultLines = append(
					resultLines,
					indentLines(arg.RequiredStatement(), 1, true),
				)
				requiredCount++
			}
		}

		if requiredCount > 0 {
			resultLines = append(resultLines, "")
		}
	}

	for o, objDef := range objsDefs {
		if o > 0 {
			resultLines = append(resultLines, "")
		}
		resultLines = append(
			resultLines,
			fmt.Sprintf("  %s = %s", objNames[o], indentLines(objDef, 1, false)),
		)
	}

	resultLines = append(resultLines, "")
	resultLines = append(resultLines, "  return [")

	for o := 0; o < len(objsDefs); o++ {
		resultLines = append(
			resultLines,
			fmt.Sprintf("%s%s,", indentToLevel(2), objNames[o]),
		)
	}

	resultLines = append(resultLines, "  ]")

	return strings.Join(resultLines, "\n"), nil
}

// walkObj recursively "walks" a reflected proto object to generate a starlark
// string representation of it.
func walkObj(
	val reflect.Value,
	level int,
	usedModules map[string]struct{},
	config Config,
	out io.Writer,
) error {
	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}

	switch val.Kind() {
	case reflect.String:
		str := val.String()

		subVar := config.SubVariable(str)
		if subVar != "" {
			// Use the variable name instead of the actual string value
			fmt.Fprint(out, subVar)
		} else if strings.ContainsAny(str, "\n") {
			// Use starlark multi-string format
			fmt.Fprintf(out, `"""%s"""`, val)
		} else {
			// Use the quoted encoding of the string
			fmt.Fprintf(out, strconv.Quote(str))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(out, "%d", val.Int())
	case reflect.Slice:
		fmt.Fprintf(out, "[\n")

		for i := 0; i < val.Len(); i++ {
			fmt.Fprintf(out, indentToLevel(level+1))

			err := walkObj(val.Index(i), level+1, usedModules, config, out)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, ",\n")
		}

		fmt.Fprintf(out, indentToLevel(level))
		fmt.Fprintf(out, "]")
	case reflect.Map:
		fmt.Fprintf(out, "{\n")

		keys := val.MapKeys()
		for _, key := range keys {
			mapValue := val.MapIndex(key)
			fmt.Fprintf(out, indentToLevel(level+1))

			err := walkObj(key, level+1, usedModules, config, out)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, ": ")

			err = walkObj(mapValue, level+1, usedModules, config, out)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, ",\n")
		}

		fmt.Fprintf(out, indentToLevel(level))
		fmt.Fprintf(out, "}")
	case reflect.Bool:
		if val.Bool() {
			fmt.Fprintf(out, "True")
		} else {
			fmt.Fprintf(out, "False")
		}
	case reflect.Struct:
		// Quantity and IntOrString are special types because we need
		// to wrap them in our special util functions.
		if val.Type().Name() == "Quantity" {
			q := val.Interface().(resource.Quantity)
			fmt.Fprintf(out, "util.quantity(\"%s\")", q.String())
			return nil
		} else if val.Type().Name() == "IntOrString" {
			q := val.Interface().(intstr.IntOrString)
			if q.IntVal > 0 {
				fmt.Fprintf(out, "util.intOrStr(%d)", q.IntVal)
			} else {
				fmt.Fprintf(out, "util.intOrStr(\"%s\")", q.StrVal)
			}

			return nil
		}

		module, err := PkgToModule(val.Type().PkgPath())
		if err != nil {
			return err
		}

		fullName := fmt.Sprintf("%s.%s", module, val.Type().Name())
		usedModules[module] = struct{}{}

		fmt.Fprintf(out, "%s(\n", fullName)

		for f := 0; f < val.NumField(); f++ {
			field := val.Type().Field(f)
			fieldValue := val.Field(f)

			// Don't include type meta in output since we automatically add
			// this when evaluating starlark
			if field.Name == "TypeMeta" {
				continue
			}

			if !fieldValue.IsZero() {
				tagValue := field.Tag.Get("json")
				if tagValue == "" {
					continue
				}

				components := strings.Split(tagValue, ",")

				if components[0] != "" {
					// Have an explicit JSON name, use this as the starlark field name
					fmt.Fprintf(out, indentToLevel(level+1))
					fmt.Fprintf(out, "%s = ", components[0])
				} else if len(components) == 2 && components[1] == "inline" {
					// Don't have json name, just lowercase the field name
					fieldName := field.Name

					fieldName = fmt.Sprintf(
						"%s%s",
						strings.ToLower(fieldName[0:1]),
						fieldName[1:],
					)

					fmt.Fprintf(out, indentToLevel(level+1))
					fmt.Fprintf(out, "%s = ", fieldName)
				} else {
					continue
				}

				err := walkObj(fieldValue, level+1, usedModules, config, out)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, ",\n")
			}
		}

		fmt.Fprintf(out, indentToLevel(level))
		fmt.Fprintf(out, ")")
	default:
		return fmt.Errorf(
			"Got unrecognized kind: %s (type: %s)",
			val.Kind(),
			val.Type(),
		)
	}

	return nil
}

func indentToLevel(level int) string {
	components := []string{}

	for l := 0; l < level; l++ {
		// Indent using 2 spaces per level
		components = append(components, "  ")
	}

	return strings.Join(components, "")
}

func indentLines(input string, level int, indentFirst bool) string {
	lines := strings.Split(input, "\n")

	var multilineQuote bool

	for l := 0; l < len(lines); l++ {
		if l == 0 && !indentFirst {
			continue
		}

		if !multilineQuote {
			// Only indent if we're not inside a multiline quote
			lines[l] = fmt.Sprintf("%s%s", indentToLevel(level), lines[l])
		}

		if strings.Contains(lines[l], `"""`) {
			multilineQuote = !multilineQuote
		}
	}

	return strings.Join(lines, "\n")
}

func objName(obj runtime.Object, existingNames map[string]struct{}) string {
	kubeObj, ok := obj.(metav1.Object)

	if ok {
		var prefix string

		if kubeObj.GetName() != "" {
			prefix = strings.ReplaceAll(
				strings.ToLower(kubeObj.GetName()),
				"-",
				"_",
			)
			prefix += "_"
		}

		kind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)

		for i := 0; i < len(existingNames)+1; i++ {
			var indexedName string

			if i == 0 {
				indexedName = fmt.Sprintf("%s%s", prefix, kind)
			} else {
				indexedName = fmt.Sprintf("%s%s%d", prefix, kind, i)
			}

			if _, ok := existingNames[indexedName]; !ok {
				return indexedName
			}
		}
	}

	return fmt.Sprintf("obj%d", len(existingNames))
}
