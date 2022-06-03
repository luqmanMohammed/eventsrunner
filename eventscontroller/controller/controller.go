// Package controller will be responsible to react to events creation and deletion.
package controller

import (
	erapi "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
	erinformers "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/internalinterfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Controller is responsible for listening for new event crds and reacting to their
// creation and deletion.
type Controller struct {
	namespace       string
	name            string
	informerFactory erinformers.SharedInformerFactory
}

// New creates a new Controller instance and returns the pointer to it.
func New(kubeconfig *rest.Config, name, namespace string) (*Controller, error) {
	clientSet, err := versioned.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	inf := erinformers.NewFilteredSharedInformerFactory(clientSet, 0, namespace, internalinterfaces.TweakListOptionsFunc(func(lo *metav1.ListOptions) {
		lo.LabelSelector = "eventsrunner.io/controller=" + name
	}))
	eventInformer := inf.Eventsrunner().V1alpha1().Events()
	eventInformer.Informer().AddIndexers(cache.Indexers{
		"rule": func(obj interface{}) ([]string, error) {
			event := obj.(*erapi.Event)
			return []string{event.Spec.RuleID}, nil
		},
		"resource": func(obj interface{}) ([]string, error) {
			event := obj.(*erapi.Event)
			return []string{event.Spec.ResourceID}, nil
		},
	})
	return &Controller{
		name:            name,
		namespace:       namespace,
		informerFactory: inf,
	}, nil
}




// Run will start the controller.
func (Controller) Run() {

}
