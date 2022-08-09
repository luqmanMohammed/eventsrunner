package helpers_test

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/helpers"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/index"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

//vscode-gox4vTmq

const (
	testNamespace = "default"
	testLabel     = "eventsrunner.io/controller=helper-test"
	labelKey      = "eventsrunner.io/controller"
	labelValue    = "helper-test"
)

var ()

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var mgr ctrl.Manager
var compHelper helpers.CompositeHelper

var cancelFunc context.CancelFunc

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = eventsrunneriov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:           scheme.Scheme,
		Port:             9443,
		LeaderElection:   false,
		LeaderElectionID: "322188aa.eventsrunner.io",
	})
	Expect(err).NotTo(HaveOccurred())

	var mgrContext context.Context
	mgrContext, cancelFunc = context.WithCancel(context.Background())

	err = index.RegisterIndexes(context.Background(), mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		err := mgr.Start(mgrContext)
		Expect(err).NotTo(HaveOccurred())
	}()

	time.Sleep(2 * time.Second)

	k8sClient = mgr.GetClient()
	Expect(k8sClient).NotTo(BeNil())

	compHelper = helpers.NewCompositeHelper(testLabel, testNamespace, k8sClient)
	Expect(compHelper).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	cancelFunc()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("CompositeHelper", func() {
	Describe("Job Helper", func() {
		testRunner := &eventsrunneriov1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Spec: eventsrunneriov1alpha1.RunnerSpec(corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: testNamespace,
					Labels: map[string]string{
						labelKey: labelValue,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-runner",
							Image: "busybox",
						},
					},
				},
			}),
		}
		testRunnerBinding := &eventsrunneriov1alpha1.RunnerBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner-binding",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			RunnerName: "test-runner",
			Rules: []string{
				"test-rule",
			},
		}
		testEvent := &eventsrunneriov1alpha1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Spec: eventsrunneriov1alpha1.EventSpec{
				RuleID:     "test-rule",
				ResourceID: "test-resource",
				EventType:  eventsrunneriov1alpha1.EventTypeAdded,
			},
		}

		When("Resolving Runner", func() {
			It("Ideal workflow", func() {
				err := k8sClient.Create(context.Background(), testRunner)
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Create(context.Background(), testRunnerBinding)
				Expect(err).NotTo(HaveOccurred())
				runnerName, err := compHelper.ResolveRunner(context.Background(), testEvent)
				Expect(err).NotTo(HaveOccurred())
				Expect(runnerName).NotTo(BeEmpty())
				Expect(runnerName).To(Equal("test-runner"))

			})
			It("Runner binding not found", func() {
				testEvent.Spec.RuleID = "test-rule-2"
				runnerName, err := compHelper.ResolveRunner(context.Background(), testEvent)
				Expect(err).To(HaveOccurred())
				Expect(runnerName).To(BeEmpty())
			})
			It("Multiple Runner bindings found", func() {
				testRunnerBinding.ObjectMeta.Name = "test-runner-binding-2"
				testRunnerBinding.ResourceVersion = ""
				testEvent.Spec.RuleID = "test-rule"
				err := k8sClient.Create(context.Background(), testRunnerBinding)
				Expect(err).NotTo(HaveOccurred())
				runnerName, err := compHelper.ResolveRunner(context.Background(), testEvent)
				Expect(err).NotTo(HaveOccurred())
				Expect(runnerName).NotTo(BeEmpty())
			})
		})
		When("Getting a resolved Runner", func() {
			It("Ideal workflow", func() {
				runner, err := compHelper.GetRunner(context.Background(), "test-runner")
				Expect(err).NotTo(HaveOccurred())
				Expect(runner).NotTo(BeNil())
				Expect(runner.Name).To(Equal("test-runner"))
			})
			It("Runner not found", func() {
				runner, err := compHelper.GetRunner(context.Background(), "test-runner-2")
				Expect(err).To(HaveOccurred())
				Expect(runner).To(BeNil())
			})
		})
	})

	Describe("Events Helper", func() {
		testEvent := &eventsrunneriov1alpha1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Spec: eventsrunneriov1alpha1.EventSpec{
				RuleID:     "test-rule",
				ResourceID: "test-resource",
				EventType:  eventsrunneriov1alpha1.EventTypeAdded,
			},
		}

		When("Finds if an event depends on any other event", func() {
			It("Ideal workflow", func() {
				for i := 0; i < 10; i++ {
					testEvent.ObjectMeta.Name = fmt.Sprintf("test-event-%d", i)
					testEvent.ResourceVersion = ""
					testEvent.CreationTimestamp = metav1.Time{}
					err := k8sClient.Create(context.Background(), testEvent)
					Expect(err).NotTo(HaveOccurred())
				}

				time.Sleep(time.Second)

				dependsOn, err := compHelper.FindEventDependsOn(context.Background(), testEvent)
				Expect(err).NotTo(HaveOccurred())
				Expect(dependsOn).NotTo(BeNil())
				fmt.Println(dependsOn.Name)
			})
		})
	})
})
