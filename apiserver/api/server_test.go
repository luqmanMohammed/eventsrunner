package api

import (
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

// TODO: Move common code to a common module
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

func TestServerStartup(t *testing.T) {
	serverOpts := ServerOpts{
		Addr:        "0.0.0.0",
		Port:        8080,
		HealthzPort: 8081,
		Namespace:   "eventsrunner",
		AuthType:    None,
		EnableTLS:   false,
	}

	config, err := GetKubeAPIConfig("")
	if err != nil {
		t.Fatalf("Error getting rest config: %v", err)
	}

	runnerAPI, err := NewEventsRunnerAPI(config, serverOpts)
	if err != nil {
		t.Fatalf("Error creating runner API: %v", err)
	}
	runnerAPI.Start()

}
