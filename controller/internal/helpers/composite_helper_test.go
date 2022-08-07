package helpers_test

import (
	"context"
	"path/filepath"

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

var (
	testRunner *eventsrunneriov1alpha1.Runner = &eventsrunneriov1alpha1.Runner{
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
	testRunnerBinding *eventsrunneriov1alpha1.RunnerBinding = &eventsrunneriov1alpha1.RunnerBinding{
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
	testEvent *eventsrunneriov1alpha1.Event = &eventsrunneriov1alpha1.Event{
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
)

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
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	Expect(err).NotTo(HaveOccurred())

	var mgrContext context.Context
	mgrContext, cancelFunc = context.WithCancel(context.Background())

	go func() {
		err := mgr.Start(mgrContext)
		Expect(err).NotTo(HaveOccurred())
	}()

	err = index.RegisterIndexes(context.Background(), mgr)
	Expect(err).NotTo(HaveOccurred())

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
		Describe("ResolveRunner", func() {
			It("Setting up objects", func() {
				err := k8sClient.Create(context.Background(), testRunner)
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Create(context.Background(), testRunnerBinding)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Resolve Runner", func() {
				runnerName, err := compHelper.ResolveRunner(context.Background(), testEvent)
				Expect(err).NotTo(HaveOccurred())
				Expect(runnerName).NotTo(BeEmpty())
				Expect(runnerName).To(Equal("test-runner"))
			})
		})
	})
})
