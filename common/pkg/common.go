package pkg

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

// GetKubeAPIConfig returns a kubeconfig regardless of in-cluster or out-of-cluster
// If running in-cluster, it will use the in-cluster config
// If running out-of-cluster, it will use config at provided path, or the default path
func GetKubeAPIConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath == "" {
		klog.V(3).Info("Provided KubeConfig path is empty. Getting config from home")
		if home := homedir.HomeDir(); home != "" {
			kubeConfigPath = home + "/.kube/config"
		}
	}
	clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return clientcmd.BuildConfigFromFlags("", "")
	}
	return clientConfig, nil
}

// GetKubeAPIConfigOrDie wraps GetKubeAPIConfig and panics if it fails
func GetKubeAPIConfigOrDie(kubeConfigPath string) *rest.Config {
	clientConfig, err := GetKubeAPIConfig(kubeConfigPath)
	if err != nil {
		klog.Fatalf("Failed to get kube config: %v", err)
	}
	return clientConfig
}

// ConvertInterfaceSliceToTyped converts slice of interface{} to a slice of concrete type
// provided via generics type parameter
func ConvertInterfaceSliceToTyped[T any](slice []interface{}) []T {
	retSlice := make([]T, len(slice))
	for i, v := range slice {
		retSlice[i] = v.(T)
	}
	return retSlice
}

// K8sTimeSorter is an interface that can be implemented by K8s objects that have a creation timestamp
type K8sTimeSorter interface {
	GetCreationTimestamp() metav1.Time
}

// SortK8sObjectsSliceByCreationTimestamp sorts a slice of K8s objects by creation timestamp
func SortK8sObjectsSliceByCreationTimestamp[T K8sTimeSorter](slice []T) {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].GetCreationTimestamp().UTC().Before(
			slice[j].GetCreationTimestamp().UTC())
	})
}
