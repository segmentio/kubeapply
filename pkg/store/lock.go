package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	coordv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/tools/clientcmd"

	// Note: Using a local fork that includes fix in
	// https://github.com/kubernetes/kubernetes/pull/85474
	"github.com/segmentio/kubeapply/pkg/store/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	kubeLockerReleaseTimeout = 10 * time.Second
)

// Locker is an interface for structs that can acquire and release locks.
type Locker interface {
	// Acquire acquires the lock with the provided name.
	Acquire(ctx context.Context, name string) error

	// Release releases the lock with the provided name.
	Release(name string) error
}

var _ Locker = (*LocalLocker)(nil)
var _ Locker = (*KubeLocker)(nil)

// LocalLocker is an implementation of Locker that keeps track of locks in memory. For testing
// purposes only.
type LocalLocker struct {
	lock sync.Mutex
	held map[string]struct{}
}

// NewLocalLocker returns a new LocalLocker instance, which is backed by a golang lock.
func NewLocalLocker() *LocalLocker {
	return &LocalLocker{
		held: map[string]struct{}{},
	}
}

// Acquire acquires the lock with the argument name.
func (l *LocalLocker) Acquire(ctx context.Context, name string) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.held[name]; ok {
		return errors.New("Lock already held")
	}

	l.held[name] = struct{}{}

	return nil
}

// Release releases the lock with the argument name.
func (l *LocalLocker) Release(name string) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.held[name]; !ok {
		return errors.New("Lock was not held")
	}

	delete(l.held, name)
	return nil
}

// KubeLocker is an Locker that uses Kubernetes's leader election functionality for locking.
type KubeLocker struct {
	id                 string
	namespace          string
	objLock            sync.Mutex
	lockCancellations  map[string]context.CancelFunc
	coordinationClient coordv1.CoordinationV1Interface
	done               chan struct{}
}

// NewKubeLocker returns a Locker that is backed by a lock in Kubernetes.
func NewKubeLocker(
	kubeConfigPath string,
	id string,
	namespace string,
) (*KubeLocker, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	coordinationClient := client.CoordinationV1()
	done := make(chan struct{}, 1)

	return &KubeLocker{
		id:                 id,
		namespace:          namespace,
		lockCancellations:  map[string]context.CancelFunc{},
		coordinationClient: coordinationClient,
		done:               done,
	}, nil
}

// Acquire acquires a lock with the argument name.
func (k *KubeLocker) Acquire(ctx context.Context, name string) error {
	log.Infof("Acquiring lock with name %s", name)

	k.objLock.Lock()
	if _, ok := k.lockCancellations[name]; ok {
		return fmt.Errorf("Lock already acquired for name %s", name)
	}
	// Create a separate context for the lock itself
	lockCtx, lockCancel := context.WithCancel(context.Background())
	k.lockCancellations[name] = lockCancel
	k.objLock.Unlock()

	leaseName := fmt.Sprintf("kubeapply-lock-%s", name)

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: k.namespace,
		},
		Client: k.coordinationClient,
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: k.id,
		},
	}

	acquired := make(chan struct{}, 1)

	le, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   20 * time.Second,
			RenewDeadline:   10 * time.Second,
			RetryPeriod:     5 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					log.Infof("Starting leading for lease %s", leaseName)
					acquired <- struct{}{}
				},
				OnStoppedLeading: func() {
					log.Warn("Lock lost")
					k.done <- struct{}{}
				},
			},
		},
	)
	if err != nil {
		return err
	}

	go le.Run(lockCtx)

	select {
	case <-ctx.Done():
		log.Warnf("Context cancelled, releasing lock %s", name)
		k.Release(name)
		return ctx.Err()
	case <-acquired:
		log.Info("Lock successfully acquired")
	}

	return nil
}

// Release releases the lock with the argument name.
func (k *KubeLocker) Release(name string) error {
	k.objLock.Lock()
	defer k.objLock.Unlock()

	log.Infof("Releasing lock with name %s", name)

	cancel, ok := k.lockCancellations[name]
	if !ok {
		return fmt.Errorf("Lock was not acquired for name %s", name)
	}

	cancel()
	delete(k.lockCancellations, name)

	log.Infof("Waiting for lock to be released")
	releaseCtx, releaseCancel := context.WithTimeout(
		context.Background(),
		kubeLockerReleaseTimeout,
	)
	defer releaseCancel()

	select {
	case <-k.done:
		return nil
	case <-releaseCtx.Done():
		return releaseCtx.Err()
	}
}
