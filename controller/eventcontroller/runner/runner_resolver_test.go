package runner

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/luqmanMohammed/eventsrunner/common/pkg"
	erAPI "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	erClient "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
)

func runShellCommand(t *testing.T, command string) {
	t.Helper()
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to execute command %s: %v", command, err)
	}
}

func envSetup(t *testing.T) {
	t.Helper()
	runShellCommand(t, "kubectl create namespace eventsrunner")
	runShellCommand(t, "kubectl apply -f ../../../crd/manifests")
	time.Sleep(time.Second * 5)
}

func envCleanup(t *testing.T) {
	t.Helper()
	runShellCommand(t, "kubectl delete namespace eventsrunner")
	runShellCommand(t, "kubectl delete -f ../../../crd/manifests")
}

func createRunnerCRD(t *testing.T, name string) *erAPI.Runner {
	t.Helper()
	runner, err := erClient.NewForConfigOrDie(common.GetKubeAPIConfigOrDie("")).
		EventsrunnerV1alpha1().
		Runners("eventsrunner").
		Create(context.Background(), &erAPI.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "eventsrunner",
				Labels: map[string]string{
					"eventsrunner.io/controller": "default",
				},
			},
			Spec: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "runner",
						Image: "eventsrunner/terraform-runner",
					},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return runner
}

func deleteRunnerCRD(t *testing.T, name string) {
	t.Helper()
	err := erClient.NewForConfigOrDie(common.GetKubeAPIConfigOrDie("")).
		EventsrunnerV1alpha1().
		Runners("eventsrunner").
		Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func createRunnerBindingCRD(t *testing.T, name string, runnerName string, rules []string) *erAPI.RunnerBinding {
	t.Helper()
	runnerBinding, err := erClient.NewForConfigOrDie(common.GetKubeAPIConfigOrDie("")).
		EventsrunnerV1alpha1().
		RunnerBindings("eventsrunner").
		Create(context.Background(), &erAPI.RunnerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "eventsrunner",
				Labels: map[string]string{
					"eventsrunner.io/controller": "default",
				},
			},
			Runner: runnerName,
			Rules:  rules,
			Overides: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: "eventsrunner/ansible-runner",
					},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return runnerBinding
}

func deleteRunnerBindingCRD(t *testing.T, name string) {
	t.Helper()
	err := erClient.NewForConfigOrDie(common.GetKubeAPIConfigOrDie("")).
		EventsrunnerV1alpha1().
		RunnerBindings("eventsrunner").
		Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func retryTest(t *testing.T, f func(try int) bool, retryCount int, interval time.Duration, failMsg string) {
	t.Helper()
	for i := 0; i < retryCount; i++ {
		if f(i) {
			return
		}
		time.Sleep(interval)
	}
	t.Fatal(failMsg)
}

var (
	mockEvent erAPI.Event = erAPI.Event{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: erAPI.EventSpec{
			RuleID: "rule1",
		},
	}
)

func TestRunnerResolverStart(t *testing.T) {
	envSetup(t)
	defer envCleanup(t)
	config, err := common.GetKubeAPIConfig("")
	if err != nil {
		t.Fatal(err)
	}

	createRunnerCRD(t, "runner1")
	defer deleteRunnerCRD(t, "runner1")

	createRunnerBindingCRD(t, "binding1", "runner1", []string{"rule1"})
	defer deleteRunnerBindingCRD(t, "binding1")

	createRunnerBindingCRD(t, "binding2", "runner1", []string{"*:rule2"})
	defer deleteRunnerBindingCRD(t, "binding2")

	createRunnerBindingCRD(t, "binding3", "runner1", []string{"added:rule3", "updated:rule3"})
	defer deleteRunnerBindingCRD(t, "binding3")

	runnerResolver, err := newRunnerResolver(config, "eventsrunner", "default")
	if err != nil {
		t.Fatal(err)
	}
	go runnerResolver.start()
	defer runnerResolver.stop()

	retryTest(t, func(try int) bool {
		rules, err := runnerResolver.runnerBindingsInformer.Informer().GetIndexer().ByIndex("rules", "rule1")
		if err != nil {
			t.Fatalf("failed to get rules: %v", err)
		}
		if len(rules) != 1 {
			t.Logf("try %d: expected 1 rule, got %d", try, len(rules))
			return false
		}
		found := false
		for _, rule := range rules {
			if rule.(*erAPI.RunnerBinding).Name == "binding1" {
				t.Log("binding1 found")
				found = true
				break
			}
		}
		if !found {
			t.Logf("try %d: expected binding1 to be found", try)
			return false
		}
		return true
	}, 10, time.Second, "binding1 not found")
}
