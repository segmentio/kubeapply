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

const kubeConfigTestPath = "../../.kube/kind-kubeapply-test.yaml"

func TestKubeLocker(t *testing.T) {
	if !util.KindEnabled() {
		t.Skipf("Skipping because kind is not enabled")
	}

	ctx := context.Background()

	namespace := fmt.Sprintf("test-kube-locker-%d", time.Now().UnixNano()/1000)
	util.CreateNamespace(t, ctx, namespace, kubeConfigTestPath)

	locker1, err := NewKubeLocker(kubeConfigTestPath, "client1", namespace)
	require.Nil(t, err)

	locker2, err := NewKubeLocker(kubeConfigTestPath, "client2", namespace)
	require.Nil(t, err)

	acquireCtx1, cancel1 := context.WithTimeout(ctx, time.Second)
	defer cancel1()
	err = locker1.Acquire(acquireCtx1, "test-key")
	require.Nil(t, err)

	// Can't acquire because lock is held by locker1
	acquireCtx2, cancel2 := context.WithTimeout(ctx, time.Second)
	defer cancel2()
	err = locker2.Acquire(acquireCtx2, "test-key")
	assert.NotNil(t, err)

	err = locker1.Release("test-key")
	assert.Nil(t, err)

	time.Sleep(500 * time.Millisecond)

	// Can acquire lock now because it was released by client1
	acquireCtx3, cancel3 := context.WithTimeout(ctx, time.Second)
	defer cancel3()
	err = locker1.Acquire(acquireCtx3, "test-key")
	require.Nil(t, err)
}
