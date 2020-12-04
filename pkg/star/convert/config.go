package convert

import (
	"fmt"
)

type Config struct {
	Entrypoint string
	Args       []Arg

	varSubs map[string]string
}

type Arg struct {
	Name         string
	DefaultValue interface{}
	Required     bool
}

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
