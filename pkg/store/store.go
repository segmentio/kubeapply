package store

import (
	"context"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Store is an interface for structs that can get and set key/value pairs.
type Store interface {
	// Get gets the value of the argument key.
	Get(ctx context.Context, key string) (string, error)

	// Set sets the provided key/value pair.
	Set(ctx context.Context, key string, value string) error
}

var _ Store = (*InMemoryStore)(nil)
var _ Store = (*KubeStore)(nil)

// InMemoryStore is an implementation of Store that is backed by a golang map. For testing
// purposes only.
type InMemoryStore struct {
	valuesMap map[string]string
}

// NewInMemoryStore returns a new InMemoryStore instance.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		valuesMap: map[string]string{},
	}
}

// Get returns the value of the argument key.
func (s *InMemoryStore) Get(ctx context.Context, key string) (string, error) {
	return s.valuesMap[key], nil
}

// Set sets the argument key to the argument value.
func (s *InMemoryStore) Set(ctx context.Context, key string, value string) error {
	s.valuesMap[key] = value
	return nil
}

// KubeStore is an implementation of Store that is backed by a Kubernetes configMap.
type KubeStore struct {
	name            string
	namespace       string
	configMapClient v1.ConfigMapInterface
}

// NewKubeStore returns a new KubeStore instance.
func NewKubeStore(
	kubeConfigPath string,
	name string,
	namespace string,
) (*KubeStore, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	configMapClient := client.CoreV1().ConfigMaps(namespace)

	return &KubeStore{
		name:            name,
		namespace:       namespace,
		configMapClient: configMapClient,
	}, nil
}

// Get returns the value of the argument key.
func (k *KubeStore) Get(ctx context.Context, key string) (string, error) {
	configMap, err := k.configMapClient.Get(ctx, k.name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Infof(
			"Could not find configmap %s in namespace %s",
			k.name,
			k.namespace,
		)
		return "", nil
	} else if err != nil {
		return "", err
	}

	if configMap.Data == nil {
		return "", nil
	}

	return configMap.Data[key], nil
}

// Set sets the argument key to the argument value. The key/value pair is stored
// in a ConfigMap.
func (k *KubeStore) Set(ctx context.Context, key string, value string) error {
	configMap, err := k.configMapClient.Get(ctx, k.name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Infof(
			"Could not find configmap %s in namespace %s; creating",
			k.name,
			k.namespace,
		)

		configMap, err = k.configMapClient.Create(
			ctx,
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.name,
					Namespace: k.namespace,
				},
			},
			metav1.CreateOptions{},
		)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	configMap.Data[key] = value

	_, err = k.configMapClient.Update(ctx, configMap, metav1.UpdateOptions{})
	return err
}
