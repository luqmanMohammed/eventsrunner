package runner

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	erAPI "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
	erInformers "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/internalinterfaces"
)

type runnerResolver struct {
	stopChan               chan struct{}
	runnerInformerFactory  erInformers.SharedInformerFactory
	runnerInformer         v1alpha1.RunnerInformer
	runnerBindingsInformer v1alpha1.RunnerBindingInformer
}

func newRunnerResolver(kubeconfig *rest.Config, namespace string, controllerName string) (*runnerResolver, error) {
	clientset, err := versioned.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	inf := erInformers.NewFilteredSharedInformerFactory(clientset, 0, namespace, internalinterfaces.TweakListOptionsFunc(func(lo *metav1.ListOptions) {
		lo.LabelSelector = "eventsrunner.io/controller=" + controllerName
	}))
	runnerInformer := inf.Eventsrunner().V1alpha1().Runners()
	runnerBindingInformer := inf.Eventsrunner().V1alpha1().RunnerBindings()

	runnerInformer.Informer().AddIndexers(cache.Indexers{
		"name": cache.MetaNamespaceIndexFunc,
	})

	runnerBindingInformer.Informer().AddIndexers(cache.Indexers{
		"rules": func(obj interface{}) ([]string, error) {
			binding := obj.(*erAPI.RunnerBinding)
			return binding.Rules, nil
		},
	})

	return &runnerResolver{
		runnerInformerFactory:  inf,
		runnerInformer:         runnerInformer,
		runnerBindingsInformer: runnerBindingInformer,
	}, nil
}

func (r *runnerResolver) start() {
	r.stopChan = make(chan struct{})
	r.runnerInformerFactory.Start(r.stopChan)
	r.runnerInformerFactory.WaitForCacheSync(r.stopChan)
	<-r.stopChan
}

func (r *runnerResolver) stop() {
	close(r.stopChan)
}
