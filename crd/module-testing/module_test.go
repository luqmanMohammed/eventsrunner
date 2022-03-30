package moduletesting

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	eralphav1 "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	eralphav1client "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned/typed/eventsrunner.io/v1alpha1"
	v1 "k8s.io/api/core/v1"
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
	
	t.Run("Test Event CRD", func(t *testing.T) {
		// List Initial Events - Should be 0 since no events have been created
		eventsList, err := erClient.Events("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing events: %v", err)
		}
		if len(eventsList.Items) != 0 {
			t.Fatalf("Expected 0 events, got %d", len(eventsList.Items))
		}
		// Create Event - Should be successful
		testEREvent := &eralphav1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event",
				Namespace: "default",
			},
			Spec: eralphav1.EventSpec{
				EventType:  "added",
				RuleID:     "test-rule",
				ResourceID: "test-resource",
				EventData: []string{
					"test",
					"test2",
				},
			},
		}
		_, err = erClient.Events("default").Create(context.TODO(), testEREvent, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating event: %v", err)
		}
		// Get created event and make sure it is correct
		time.Sleep(time.Second * 2)
		erEvent, err := erClient.Events("default").Get(context.TODO(), testEREvent.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Error getting event: %v", err)
		}
		if erEvent.Spec.EventType != "added" {
			t.Fatalf("Expected event type to be added, got %s", erEvent.Spec.EventType)
		}
		// Delete created event - Should be successful
		err = erClient.Events("default").Delete(context.TODO(), testEREvent.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Error deleting event: %v", err)
		}
		time.Sleep(time.Second * 2)
		// List events - Should be 0 since the event was deleted
		eventsList, err = erClient.Events("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing events: %v", err)
		}
		if len(eventsList.Items) != 0 {
			t.Fatalf("Expected 0 events, got %d", len(eventsList.Items))
		}
	})

	t.Run("Test Runner CRD", func(t *testing.T) {
		// List Initial Runners - Should be 0 since no runners have been created
		runnersList, err := erClient.Runners("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing runners: %v", err)
		}
		if len(runnersList.Items) != 0 {
			t.Fatalf("Expected 0 runners, got %d", len(runnersList.Items))
		}
		// Create Runner - Should be successful
		testRunner := &eralphav1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner",
				Namespace: "default",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "test-runner",
						Image: "test-runner",
					},
				},
			},
		}
		_, err = erClient.Runners("default").Create(context.TODO(), testRunner, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating runner: %v", err)
		}
		time.Sleep(time.Second * 2)
		// Get created runner and make sure it is correct
		runner, err := erClient.Runners("default").Get(context.TODO(), testRunner.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Error getting runner: %v", err)
		}
		if len(runner.Spec.Containers) != 1 {
			t.Fatalf("Expected 1 container, got %d", len(runner.Spec.Containers))
		}
		// Delete created runner - Should be successful
		err = erClient.Runners("default").Delete(context.TODO(), testRunner.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Error deleting runner: %v", err)
		}
		time.Sleep(time.Second * 2)
		// List runners - Should be 0 since the runner was deleted
		runnersList, err = erClient.Runners("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing runners: %v", err)
		}
		if len(runnersList.Items) != 0 {
			t.Fatalf("Expected 0 runners, got %d", len(runnersList.Items))
		}
	})

	t.Run("Test Runner Binding CRD", func(t *testing.T) {
		// List Initial Runner Bindings - Should be 0 since no runner bindings have been created
		runnerBindingsList, err := erClient.RunnerBindings("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing runner bindings: %v", err)
		}
		if len(runnerBindingsList.Items) != 0 {
			t.Fatalf("Expected 0 runner bindings, got %d", len(runnerBindingsList.Items))
		}
		// Create Runner Binding - Should be successful
		testRunnerBinding := &eralphav1.RunnerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner-binding",
				Namespace: "default",
			},
			Runner: "test-runner",
			Rules: []string{
				"test-rule",
			},
		}
		_, err = erClient.RunnerBindings("default").Create(context.TODO(), testRunnerBinding, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating runner binding: %v", err)
		}
		time.Sleep(time.Second * 2)
		// Get created runner binding and make sure it is correct
		runnerBinding, err := erClient.RunnerBindings("default").Get(context.TODO(), testRunnerBinding.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Error getting runner binding: %v", err)
		}
		if runnerBinding.Runner != "test-runner" {
			t.Fatalf("Expected runner to be test-runner, got %s", runnerBinding.Runner)
		}
		// Delete created runner binding - Should
		err = erClient.RunnerBindings("default").Delete(context.TODO(), testRunnerBinding.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("Error deleting runner binding: %v", err)
		}
		time.Sleep(time.Second * 2)
		// List runner bindings - Should be 0 since the runner binding was deleted
		runnerBindingsList, err = erClient.RunnerBindings("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Error listing runner bindings: %v", err)
		}
		if len(runnerBindingsList.Items) != 0 {
			t.Fatalf("Expected 0 runner bindings, got %d", len(runnerBindingsList.Items))
		}
	})
}
