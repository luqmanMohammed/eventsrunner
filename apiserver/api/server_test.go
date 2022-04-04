package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	erClient "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned/typed/eventsrunner.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func postEvent(body map[string]interface{}) (int, *eventPostResponseBody, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	resp, err := http.Post("http://localhost:8080/api/v1/events", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, err
	}
	var eprb eventPostResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&eprb); err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, &eprb, nil
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

func TestEventPost(t *testing.T) {
	runShellCommand(t, "kubectl create namespace eventsrunner")
	runShellCommand(t, "kubectl apply -f ../../crd/manifests")
	defer runShellCommand(t, "kubectl delete namespace eventsrunner")
	defer runShellCommand(t, "kubectl delete -f ../../crd/manifests")

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
	defer runnerAPI.Stop(context.Background())

	missingRequiredFieldEvent := map[string]interface{}{
		"eventType": "added",
		"ruleID":    "test-rule",
	}
	status, respBody, err := postEvent(missingRequiredFieldEvent)
	if err != nil {
		t.Fatalf("Error posting missing required field event: %v", err)
	}
	if status != http.StatusBadRequest {
		t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, status)
	}
	if respBody.Error != "validation error: resourceID is required" {
		t.Fatalf("Expected error message 'validation error: resourceID is required', got %s", respBody.Error)
	}

	invalidEventType := map[string]interface{}{
		"eventType":  "invalid",
		"ruleID":     "test-rule",
		"resourceID": "test-resource",
	}
	status, respBody, err = postEvent(invalidEventType)
	if err != nil {
		t.Fatalf("Error posting invalid event type: %v", err)
	}
	if status != http.StatusBadRequest {
		t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, status)
	}
	if respBody.Error != "validation error: eventType must be one of added, updated, or deleted" {
		t.Fatalf("Expected error message 'validation error: eventType must be one of added, updated, or deleted', got '%s'", respBody.Error)
	}

	validEvent := map[string]interface{}{
		"eventType":  "added",
		"ruleID":     "test-rule",
		"resourceID": "test-resource",
		"eventData": []interface{}{
			map[string]interface{}{
				"key": "test-key",
			},
		},
	}
	status, respBody, err = postEvent(validEvent)
	if err != nil {
		t.Fatalf("Error posting valid event: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, status)
	}
	if !strings.HasPrefix(respBody.ResourceName, "test-resource") {
		t.Fatalf("Expected resource name to start with 'test-resource', got '%s'", respBody.ResourceName)
	}

	config, err := GetKubeAPIConfig("")
	if err != nil {
		t.Fatalf("Error getting rest config: %v", err)
	}
	erclient, err := erClient.NewForConfig(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}
	createdEvent, err := erclient.Events("eventsrunner").Get(context.Background(), respBody.ResourceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error retrieving created event: %v", err)
	}
	if createdEvent.Spec.EventType != "added" {
		t.Fatalf("Expected eventype to be 'added'. got %v", createdEvent.Spec.EventType)
	}

}

func BenchmarkPostEvent(b *testing.B) {
	runShellCommand(&testing.T{}, "kubectl create namespace eventsrunner")
	runShellCommand(&testing.T{}, "kubectl apply -f ../../crd/manifests")
	defer runShellCommand(&testing.T{}, "kubectl delete namespace eventsrunner")
	defer runShellCommand(&testing.T{}, "kubectl delete -f ../../crd/manifests")
	erapiOpts := &EventsRunnerAPIServerOpts{
		Addr:        "0.0.0.0",
		Port:        8080,
		HealthzPort: 8081,
		Namespace:   "eventsrunner",
		AuthType:    None,
		EnableTLS:   false,
	}
	config, err := GetKubeAPIConfig("")
	if err != nil {
		b.Fatalf("Error getting rest config: %v", err)
	}
	runnerAPI, err := NewEventsRunnerAPI(config, *erapiOpts)
	if err != nil {
		b.Fatalf("Error creating runner API: %v", err)
	}
	go runnerAPI.Start()
	defer runnerAPI.Stop(context.Background())
	for {
		resp, _ := http.Get("http://localhost:8081/healthz")
		if resp != nil && resp.StatusCode == 200 {
			b.Logf("Healthz endpoint is up")
			break
		} else {
			b.Log("Waiting for server to start")
			time.Sleep(time.Second)
		}
	}
	b.ResetTimer()
	b.Logf("Running benchmark with %d events", b.N)
	for i := 0; i < b.N; i++ {
		validEvent := map[string]interface{}{
			"eventType":  "added",
			"ruleID":     "test-rule",
			"resourceID": "test-resource",
			"eventData": []interface{}{
				map[string]interface{}{
					"key": "test-key",
				},
			},
		}
		statusCode, _, err := postEvent(validEvent)
		if err != nil || statusCode != http.StatusOK {
			b.Fatalf("Error posting valid event: %v", err)
		}
	}

}
