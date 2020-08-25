package events

import (
	"fmt"
	"strings"
)

type commentType string

const (
	commentTypeCommand     commentType = "command"
	commentTypeApplyResult commentType = "applyResult"
	commentTypeOther       commentType = "other"
)

type command string

const (
	commandApply  command = "apply"
	commandDiff   command = "diff"
	commandHelp   command = "help"
	commandStatus command = "status"
)

type eventCommand struct {
	cmd   command
	args  []string
	flags map[string]string
}

func commentBodyToType(body string) commentType {
	if strings.Contains(body, "🤖 Kubeapply apply result") {
		return commentTypeApplyResult
	}

	components := strings.Split(body, " ")
	if len(components) >= 2 && components[0] == "kubeapply" {
		return commentTypeCommand
	}

	return commentTypeOther
}

func getCommand(body string) (*eventCommand, error) {
	components := strings.Split(strings.TrimSpace(body), " ")

	if len(components) < 2 {
		return nil, fmt.Errorf("Must provide at least 2 args")
	}

	commandStr := components[1]

	var cmd command

	switch commandStr {
	case "apply":
		cmd = commandApply
	case "diff":
		cmd = commandDiff
	case "help":
		cmd = commandHelp
	case "status":
		cmd = commandStatus
	default:
		return nil, fmt.Errorf("Unrecognized command: %s", commandStr)
	}

	args := []string{}
	flags := map[string]string{}

	for c := 2; c < len(components); c++ {
		component := components[c]

		if component == "" {
			continue
		} else if strings.HasPrefix(component, "--") {
			subcomponents := strings.SplitN(component, "=", 2)
			key := subcomponents[0][2:]
			var value string
			if len(subcomponents) > 1 {
				value = subcomponents[1]
			}
			flags[key] = value
		} else {
			args = append(args, component)
		}
	}

	return &eventCommand{
		cmd:   cmd,
		args:  args,
		flags: flags,
	}, nil
}
