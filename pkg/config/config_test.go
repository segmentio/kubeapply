package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckVersion(t *testing.T) {
	type testCase struct {
		config   ClusterConfig
		version  string
		expMatch bool
	}

	testCases := []testCase{
		{
			config:   ClusterConfig{},
			version:  "0.0.1",
			expMatch: true,
		},
		{
			config:   ClusterConfig{VersionConstraint: ">= 0.0.3"},
			version:  "0.0.1",
			expMatch: false,
		},
		{
			config:   ClusterConfig{VersionConstraint: ">= 0.0.3"},
			version:  "0.0.3",
			expMatch: true,
		},
		{
			config:   ClusterConfig{VersionConstraint: ">= 0.0.3"},
			version:  "0.1.0",
			expMatch: true,
		},
	}

	for _, testCase := range testCases {
		err := testCase.config.CheckVersion(testCase.version)
		if testCase.expMatch {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}
