package validation

import (
	"fmt"

	kresource "github.com/yannh/kubeconform/pkg/resource"
)

// Resource is a Kubernetes resource from a file that we want to do checks on.
type Resource struct {
	Path      string
	Contents  []byte
	Name      string
	Namespace string
	Version   string
	Kind      string

	index int
}

// MakeResource constructs a resource from a path, contents, and index.
func MakeResource(path string, contents []byte, index int) Resource {
	kResource := &kresource.Resource{
		Path:  path,
		Bytes: contents,
	}

	resource := Resource{
		Path:     path,
		Contents: contents,
		index:    index,
	}

	sig, err := kResource.Signature()
	if err == nil || sig != nil {
		resource.Name = sig.Name
		resource.Kind = sig.Kind
		resource.Namespace = sig.Namespace
		resource.Version = sig.Version
	}

	return resource
}

// PrettyName returns a pretty, compact name for a resource.
func (r Resource) PrettyName() string {
	if r.Kind != "" && r.Version != "" && r.Name != "" && r.Namespace != "" {
		// Namespaced resources
		return fmt.Sprintf("%s.%s.%s.%s", r.Kind, r.Version, r.Namespace, r.Name)
	} else if r.Kind != "" && r.Version != "" && r.Name != "" {
		// Non-namespaced resources
		return fmt.Sprintf("%s.%s.%s", r.Kind, r.Version, r.Name)
	} else {
		return r.Name
	}
}

func (r Resource) TokResource() kresource.Resource {
	return kresource.Resource{
		Path:  r.Path,
		Bytes: r.Contents,
	}
}
