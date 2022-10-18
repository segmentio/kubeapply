package convert

import (
	"fmt"
	"strings"
)

var modules = map[string]string{
	// Need to include the package of every proto type that might be
	// used.
	"appsv1":            "k8s.io/api/apps/v1",
	"appsv1beta1":       "k8s.io/api/apps/v1beta1",
	"appsv1beta2":       "k8s.io/api/apps/v1beta2",
	"authenticationv1":  "k8s.io/api/authentication/v1",
	"authorizationv1":   "k8s.io/api/authorization/v1",
	"autoscalingv1":     "k8s.io/api/autoscaling/v1",
	"batchv1":           "k8s.io/api/batch/v1",
	"batchv1beta1":      "k8s.io/api/batch/v1beta1",
	"corev1":            "k8s.io/api/core/v1",
	"eventsv1beta1":     "k8s.io/api/events/v1beta1",
	"extensionsv1beta1": "k8s.io/api/extensions/v1beta1",
	"metav1":            "k8s.io/apimachinery/pkg/apis/meta/v1",
	"networkingv1":      "k8s.io/api/networking/v1",
	"policyv1beta1":     "k8s.io/api/policy/v1beta1",
	"rbacv1":            "k8s.io/api/rbac/v1",
	"rbacv1beta1":       "k8s.io/api/rbac/v1beta1",
	"resource":          "k8s.io/apimachinery/pkg/api/resource",
	"schedulingv1":      "k8s.io/api/scheduling/v1",
	"schedulingv1beta1": "k8s.io/api/scheduling/v1beta1",
	"storagev1":         "k8s.io/api/storage/v1",
}

// PkgToModule converts a go package into the corresponding skycfg
// module name.
func PkgToModule(pkgName string) (string, error) {
	for key, value := range modules {
		if value == pkgName {
			return key, nil
		}
	}

	return "", fmt.Errorf("Could not find module for package %s", pkgName)
}

// ModuleToImportName converts a module name into the corresponding
// skycfg go package name.
func ModuleToImportName(module string) (string, error) {
	packageName, ok := modules[module]
	if !ok {
		return "", fmt.Errorf("Could not find package for %s", module)
	}

	return strings.ReplaceAll(packageName, "/", "."), nil
}
