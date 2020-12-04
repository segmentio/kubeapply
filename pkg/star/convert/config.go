package convert

import (
	"fmt"
)

// Config stores the configuration associated with a YAML to starlark conversion.
type Config struct {
	Entrypoint string
	Args       []Arg

	varSubs map[string]string
}

// Arg includes the details of an argument for the starlark entrypoint.
type Arg struct {
	Name         string
	DefaultValue interface{}
	Required     bool
}

// SubVariable determines which variable (if any) a raw string should
// be replaced with.
func (c Config) SubVariable(value string) string {
	if c.varSubs == nil {
		c.varSubs = map[string]string{}

		for _, arg := range c.Args {
			switch v := arg.DefaultValue.(type) {
			case string:
				if v != "" {
					c.varSubs[v] = arg.Name
				}
			default:
			}
		}
	}

	return c.varSubs[value]
}

// DefaultValueStr gets the default value for this argument.
func (a Arg) DefaultValueStr() string {
	switch v := a.DefaultValue.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "True"
		} else {
			return "False"
		}
	default:
		return "None"
	}
}

// RequiredStatement returns a statement that ensures that this argument
// is set.
func (a Arg) RequiredStatement() string {
	var emptyValue string

	switch a.DefaultValue.(type) {
	case string:
		emptyValue = `""`
	case int, int8, int16, int32, int64:
		emptyValue = "0"
	case float32, float64:
		emptyValue = "0.0"
	default:
		emptyValue = "None"
	}

	return fmt.Sprintf(`if %s == %s:
  fail("%s must be set to non-empty value")`, a.Name, emptyValue, a.Name)
}
