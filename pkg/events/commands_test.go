package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommentBodyToType(t *testing.T) {
	type testCase struct {
		body    string
		expType commentType
	}

	testCases := []testCase{
		{
			body:    "random comment...",
			expType: commentTypeOther,
		},
		{
			body:    "kubeapply",
			expType: commentTypeOther,
		},
		{
			body:    "kubeapply apply",
			expType: commentTypeCommand,
		},
		{
			body:    "kubeapply apply my-cluster --subpath=.",
			expType: commentTypeCommand,
		},
		{
			body:    "header 🤖 Kubeapply apply result ...\nresults",
			expType: commentTypeApplyResult,
		},
	}

	for index, testCase := range testCases {
		assert.Equal(
			t,
			testCase.expType,
			commentBodyToType(testCase.body),
			"Test case %d", index,
		)
	}
}

func TestGetCommand(t *testing.T) {
	type testCase struct {
		body       string
		expCommand *eventCommand
		expErr     bool
	}

	testCases := []testCase{
		{
			body: "kubeapply help",
			expCommand: &eventCommand{
				cmd:   commandHelp,
				args:  []string{},
				flags: map[string]string{},
			},
		},
		{
			body: "kubeapply diff",
			expCommand: &eventCommand{
				cmd:   commandDiff,
				args:  []string{},
				flags: map[string]string{},
			},
		},
		{
			body: "kubeapply status",
			expCommand: &eventCommand{
				cmd:   commandStatus,
				args:  []string{},
				flags: map[string]string{},
			},
		},
		{
			body: "  kubeapply apply arg1   arg2  arg3 --key1=value1 --key2   --key3=value3\r\n\r\n",
			expCommand: &eventCommand{
				cmd: commandApply,
				args: []string{
					"arg1",
					"arg2",
					"arg3",
				},
				flags: map[string]string{
					"key1": "value1",
					"key2": "",
					"key3": "value3",
				},
			},
		},
		{
			body:   "kubeapply",
			expErr: true,
		},
		{
			body:   "kubeapply unknown-command",
			expErr: true,
		},
	}

	for _, testCase := range testCases {
		result, err := getCommand(testCase.body)
		if testCase.expErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, testCase.expCommand, result)
		}
	}
}
