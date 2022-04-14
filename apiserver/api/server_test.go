package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

func healthzTest(t *testing.T, port int) {
	t.Helper()
	for {
		resp, _ := http.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
		if resp != nil && resp.StatusCode == 200 {
			t.Logf("Healthz endpoint is up")
			break
		} else {
			t.Log("Waiting for server to start")
			time.Sleep(time.Second)
		}
	}
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
	healthzTest(t, 8081)
	runnerAPI.Stop(context.Background())
}

func TestStopServerOnSignal(t *testing.T) {
	runnerAPI := createNewEventsRunnerAPI(t, nil)
	go runnerAPI.StopOnSignal()
	defer func() {
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
	go runnerAPI.Start()
	healthzTest(t, 8081)
}

func TestEventPost(t *testing.T) {
	runShellCommand(t, "kubectl create namespace eventsrunner")
	runShellCommand(t, "kubectl apply -f ../../crd/manifests")
	defer runShellCommand(t, "kubectl delete namespace eventsrunner")
	defer runShellCommand(t, "kubectl delete -f ../../crd/manifests")

	runnerAPI := createNewEventsRunnerAPI(t, nil)
	go runnerAPI.Start()
	healthzTest(t, 8081)
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
			"resourceID": fmt.Sprintf("test-resource-%d", i),
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

const (
	testCaPKICertScript = `#!/bin/bash
set -xe
mkdir -p /tmp/test-pki

cat <<EOF> /tmp/test-pki/csr.conf
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
C = US
ST = CA
L = CA
O = test
OU = test
CN = 127.0.0.1

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = localhost
DNS.2 = *.cluster.local
IP.1 = 127.0.0.1

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF

openssl genrsa -out /tmp/test-pki/ca.key 2048
openssl req -x509 -new -nodes -key /tmp/test-pki/ca.key -subj "/CN=127.0.0.1" -days 10000 -out /tmp/test-pki/ca.crt

openssl genrsa -out /tmp/test-pki/server.key 2048
openssl req -new -key /tmp/test-pki/server.key -out /tmp/test-pki/server.csr -config /tmp/test-pki/csr.conf
openssl x509 -req -in  /tmp/test-pki/server.csr -CA /tmp/test-pki/ca.crt -CAkey /tmp/test-pki/ca.key -CAcreateserial -out /tmp/test-pki/server.crt -days 10000 -extensions v3_ext -extfile /tmp/test-pki/csr.conf

openssl genrsa -out /tmp/test-pki/client.key 2048
openssl req -new -key /tmp/test-pki/client.key -out /tmp/test-pki/client.csr -subj "/CN=Client"
openssl x509 -req -in  /tmp/test-pki/client.csr -CA /tmp/test-pki/ca.crt -CAkey /tmp/test-pki/ca.key -CAcreateserial -out /tmp/test-pki/client.crt -days 10000

`
)

func setupTLS() error {
	if _, err := os.Stat("/tmp/test-pki"); os.IsNotExist(err) {
		cmd := exec.Command("sh", "-c", testCaPKICertScript)
		file, err := os.OpenFile(os.DevNull, os.O_RDWR, os.ModeAppend)
		if err != nil {
			return err
		}
		cmd.Stdout = file
		cmd.Stderr = file
		time.Sleep(3 * time.Second)
		return cmd.Run()
	}
	return nil
}

func createTLSConfig(caCertPath, clientKeyPath, clientCertPath string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {

		return nil, errors.New("failed to setup tls config")
	}
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	if clientCertPath != "" && clientKeyPath != "" {
		clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {

			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	return tlsConfig, nil
}

func TestMTLSAuth(t *testing.T) {
	if err := setupTLS(); err != nil {
		t.Fatalf("Error setting up TLS: %v", err)
	}
	runShellCommand(&testing.T{}, "kubectl create namespace eventsrunner")
	runShellCommand(&testing.T{}, "kubectl apply -f ../../crd/manifests")
	defer runShellCommand(&testing.T{}, "kubectl delete namespace eventsrunner")
	defer runShellCommand(&testing.T{}, "kubectl delete -f ../../crd/manifests")
	erapiOpts := &EventsRunnerAPIServerOpts{
		Addr:        "0.0.0.0",
		Port:        8085,
		HealthzPort: 8086,
		Namespace:   "eventsrunner",
		AuthType:    MTLS,
		EnableTLS:   true,
		CACert:      "/tmp/test-pki/ca.crt",
		CertPath:    "/tmp/test-pki/server.crt",
		KeyPath:     "/tmp/test-pki/server.key",
	}
	config, err := GetKubeAPIConfig("")
	if err != nil {
		t.Fatalf("Error getting rest config: %v", err)
	}
	runnerAPI, err := NewEventsRunnerAPI(config, *erapiOpts)
	if err != nil {
		t.Fatalf("Error creating runner API: %v", err)
	}
	go runnerAPI.Start()
	defer runnerAPI.Stop(context.Background())
	healthzTest(t, 8086)

	clientTLSConfig, err := createTLSConfig("/tmp/test-pki/ca.crt", "/tmp/test-pki/client.key", "/tmp/test-pki/client.crt")
	if err != nil {
		t.Fatalf("Error creating client TLS config: %v", err)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
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
	bodyBytes, err := json.Marshal(validEvent)
	if err != nil {
		t.Fatalf("MTLS auth test failed due to %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "http://localhost:8085/api/v1/events", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("MTLS auth test failed due to %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("MTLS auth test failed due to %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200 got %v", resp.StatusCode)
	}

	var eprb eventPostResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&eprb); err != nil {
		t.Fatalf("MTLS auth test failed due to %v", err)
	}
	if eprb.Message != "Event created successfully" {
		t.Fatalf("Expected message 'Event created successfully' got %v", eprb.Message)
	}
}
