package api

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

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

func createNewEventsRunnerAPI(t *testing.T, erapiOpts *EventsRunnerAPIServerOpts) *EventsRunnerAPI {
	t.Helper()
	if erapiOpts == nil {
		erapiOpts = &EventsRunnerAPIServerOpts{
			Addr:        "0.0.0.0",
			Port:        8080,
			HealthzPort: 8081,
			Namespace:   "eventsrunner",
			AuthType:    None,
			EnableTLS:   false,
		}
	}
	config, err := GetKubeAPIConfig("")
	if err != nil {
		t.Fatalf("Error getting rest config: %v", err)
	}
	runnerAPI, err := NewEventsRunnerAPI(config, *erapiOpts)
	if err != nil {
		t.Fatalf("Error creating runner API: %v", err)
	}
	return runnerAPI
}

func runShellCommand(t *testing.T, command string) {
	t.Helper()
	cmd := exec.Command("bash", "-c", command)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to execute command %s: %v", command, err)
	}
}

func TestServerStartupAndShutdown(t *testing.T) {
	runnerAPI := createNewEventsRunnerAPI(t, nil)
	go runnerAPI.Start()
	for {
		resp, _ := http.Get("http://localhost:8081/healthz")
		if resp != nil && resp.StatusCode == 200 {
			t.Logf("Healthz endpoint is up")
			break
		} else {
			t.Log("Waiting for server to start")
			time.Sleep(time.Second)
		}
	}
	runnerAPI.Stop(context.Background())
}

func TestStopServerOnSignal(t *testing.T) {
	runnerAPI := createNewEventsRunnerAPI(t, nil)
	go runnerAPI.StopOnSignal()
	defer func() {
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	go runnerAPI.Start()
	for {
		resp, _ := http.Get("http://localhost:8081/healthz")
		if resp != nil && resp.StatusCode == 200 {
			t.Logf("Healthz endpoint is up")
			break
		} else {
			t.Log("Waiting for server to start")
			time.Sleep(time.Second)
		}
	}
}
