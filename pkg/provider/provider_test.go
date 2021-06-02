package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	err := Provider(nil).InternalValidate()
	require.NoError(t, err)
}
