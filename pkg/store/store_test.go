package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubeStore(t *testing.T) {
	if !util.KindEnabled() {
		t.Skipf("Skipping because kind is not enabled")
	}

	ctx := context.Background()

	namespace := fmt.Sprintf("test-kube-store-%d", time.Now().UnixNano()/1000)
	util.CreateNamespace(t, ctx, namespace, kubeConfigTestPath)

	store, err := NewKubeStore(
		kubeConfigTestPath,
		"test-store",
		namespace,
	)
	require.Nil(t, err)

	err = store.Set("test-key", "test-value")
	require.Nil(t, err)

	result, err := store.Get("test-key")
	require.Nil(t, err)
	assert.Equal(t, "test-value", result)

	result, err = store.Get("non-existent-key")
	require.Nil(t, err)
	assert.Equal(t, "", result)
}
