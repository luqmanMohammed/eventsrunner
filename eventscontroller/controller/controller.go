// Package controller will be responsible to react to events creation and deletion.
package controller

import (
	"sort"

	common "github.com/luqmanMohammed/eventsrunner/common/pkg"
	erapi "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
	erinformers "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/internalinterfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type K8sTimeSorter interface {
	GetCreationTimestamp() metav1.Time
}

// SortK8sObjectsSliceByCreationTimestamp sorts a slice of K8s objects by creation timestamp
func SortK8sObjectsSliceByCreationTimestamp[T K8sTimeSorter](slice []T) {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].GetCreationTimestamp().UTC().Before(
			slice[j].GetCreationTimestamp().UTC())
	})
}

// Controller is responsible for listening for new event crds and reacting to their
// creation and deletion.
type Controller struct {
	namespace       string
	name            string
	informerFactory erinformers.SharedInformerFactory
	eventInformer   v1alpha1.EventInformer
}

func (c *Controller) processWaitingEvent(event *erapi.Event) {
	eventsInter, err := c.eventInformer.Informer().GetIndexer().ByIndex("resource", event.Spec.ResourceID)
	if err != nil {
		return
	}
	events := common.ConvertInterfaceSliceToTyped[*erapi.Event](eventsInter)
	if len(events) == 0 {
		event.Status.State = erapi.EventStatePending
	} else {
		SortK8sObjectsSliceByCreationTimestamp[*erapi.Event](events)
		lastEvent := events[len(events)-1]
		event.Status.DependentEventID = lastEvent.Name
	}
}

func (c *Controller) processEvent(event *erapi.Event) {
	switch event.Status.State {
	case erapi.EventStateWaiting:
		c.processWaitingEvent(event)
	case erapi.EventStateProcessing:

	}
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
