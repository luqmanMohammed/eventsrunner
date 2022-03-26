package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	eralphav1 "github.com/luqmanMohammed/eventsrunner/ereventcrd/pkg/apis/eventsrunner.io/v1alpha1"
	eralphav1client "github.com/luqmanMohammed/eventsrunner/ereventcrd/pkg/client/clientset/versioned/typed/eventsrunner.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

// GetKubeAPIConfig returns a Kubernetes API config.
// Common abstraction to get config in both incluster and out cluster
// scenarios.
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

func runShellCommand(t *testing.T, command string) {
	t.Helper()
	cmd := exec.Command("bash", "-c", command)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to execute command %s: %v", command, err)
	}
}

func TestModule(t *testing.T) {
	restConfig, err := GetKubeAPIConfig("")
	if err != nil {
		t.Fatalf("Error getting rest config: %v", err)
	}

	runShellCommand(t, "kubectl apply -f ../manifests/")
	defer runShellCommand(t, "kubectl delete -f ../manifests/")

	erClient, err := eralphav1client.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	erList, err := erClient.EREvents("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing events: %v", err)
	}

	if len(erList.Items) != 0 {
		t.Fatalf("Expected 0 events, got %d", len(erList.Items))
	}

	testEREvent := &eralphav1.EREvent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		Spec: eralphav1.EREventSpec{
			EventType:  "added",
			RuleID:     "test-rule",
			ResourceID: "test-resource",
			EventData: []map[string]string{
				{
					"key": "value",
				},
			},
		},
	}

	_, err = erClient.EREvents("default").Create(context.TODO(), testEREvent, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating event: %v", err)
	}

	time.Sleep(time.Second * 2)

	erEvent, err := erClient.EREvents("default").Get(context.TODO(), testEREvent.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting event: %v", err)
	}
	if erEvent.Spec.EventType != "added" {
		t.Fatalf("Expected event type to be added, got %s", erEvent.Spec.EventType)
	}

	erClient.EREvents("default").Delete(context.TODO(), testEREvent.Name, metav1.DeleteOptions{})

	time.Sleep(time.Second * 2)

	erList, err = erClient.EREvents("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing events: %v", err)
	}
	if len(erList.Items) != 0 {
		t.Fatalf("Expected 0 events, got %d", len(erList.Items))
	}
}
