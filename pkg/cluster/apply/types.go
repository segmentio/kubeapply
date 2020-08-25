package apply

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TypedKubeObj is a typed object that is returned by the kube API. We're using
// this instead of the formal types from the kube API because it's more flexible
// and doesn't require registering all types in a scheme, etc.
type TypedKubeObj struct {
	APIVersion   string `json:"apiVersion"`
	Kind         string `json:"kind"`
	KubeMetadata `json:"metadata"`
	Items        []TypedKubeObj `json:"items"`
}

// KubeMetadata is basic metadata about an object or list.
type KubeMetadata struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	ResourceVersion   string `json:"resourceVersion"`
	CreationTimestamp string `json:"creationTimestamp"`
}

// Result represents the result of running "kubectl apply" for a single manifest.
type Result struct {
	Name       string
	Namespace  string
	Kind       string
	CreatedAt  time.Time
	OldVersion string
	NewVersion string

	index int
}

// IsCreated returns whether this result involved a resource creation.
func (r Result) IsCreated() bool {
	return r.OldVersion == ""
}

// IsUpdated returns whether this result involved updating an existing resource.
func (r Result) IsUpdated() bool {
	return r.OldVersion != "" && r.OldVersion != r.NewVersion
}

// CreatedTimestamp returns the creation time of the resource associated with this result.
func (r Result) CreatedTimestamp() string {
	if r.IsCreated() {
		return ""
	}

	return r.CreatedAt.UTC().Format(time.RFC3339)
}

// TypedObj is an interface used for extracting metadata from Kubernetes objects.
type TypedObj interface {
	metav1.Object
	runtime.Object
}
