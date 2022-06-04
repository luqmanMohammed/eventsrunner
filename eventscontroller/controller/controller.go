// Package controller will be responsible to react to events creation and deletion.
package controller

import (
	"context"
	"fmt"

	common "github.com/luqmanMohammed/eventsrunner/common/pkg"
	erapi "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
	erinformers "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions"
	versionederinformers "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/internalinterfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Controller is responsible for listening for new event crds and reacting to their
// creation and deletion.
type Controller struct {
	namespace             string
	name                  string
	informerFactory       erinformers.SharedInformerFactory
	eventInformer         versionederinformers.EventInformer
	runnerBindingInformer versionederinformers.RunnerBindingInformer
	processQueue          chan *erapi.Event
	erClientset           *versioned.Clientset
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
		common.SortK8sObjectsSliceByCreationTimestamp[*erapi.Event](events)
		lastEvent := events[len(events)-1]
		event.Status.DependentEventID = lastEvent.Name
	}
}

func (c *Controller) processPendingEvent(event *erapi.Event) {
	runnerBindingsInter, err := c.runnerBindingInformer.Informer().GetIndexer().ByIndex("rules", event.Spec.RuleID)
	if err != nil {
		return
	}
	if len(runnerBindingsInter) == 0 {
		return
	}
	runnerBindings := common.ConvertInterfaceSliceToTyped[*erapi.RunnerBinding](runnerBindingsInter)
	common.SortK8sObjectsSliceByCreationTimestamp[*erapi.RunnerBinding](runnerBindings)
	lastRunnerBinding := runnerBindings[len(runnerBindings)-1]
	event.Status.RuleBindingName = lastRunnerBinding.Name
	event.Status.RunnerName = lastRunnerBinding.Runner
	event.Status.State = erapi.EventStateAssigned
}

func (c *Controller) processEvent(event *erapi.Event) {
	fmt.Printf("Processing event %s\n", event.Name)
	switch event.Status.State {
	case erapi.EventStateEmpty:
		event.Status.State = erapi.EventStateWaiting
	case erapi.EventStateWaiting:
		c.processWaitingEvent(event)
	case erapi.EventStatePending:
		c.processPendingEvent(event)
	case erapi.EventStateAssigned:
	default:
		return
	}
	_, err := c.erClientset.EventsrunnerV1alpha1().Events(c.namespace).Update(context.TODO(), event, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println(err)
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
	runnerBindingInformer := inf.Eventsrunner().V1alpha1().RunnerBindings()
	runnerBindingInformer.Informer().AddIndexers(cache.Indexers{
		"rules": func(obj interface{}) ([]string, error) {
			binding := obj.(*erapi.RunnerBinding)
			return binding.Rules, nil
		},
	})

	clientset, err := versioned.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &Controller{
		name:                  name,
		namespace:             namespace,
		informerFactory:       inf,
		eventInformer:         eventInformer,
		runnerBindingInformer: runnerBindingInformer,
		processQueue:          make(chan *erapi.Event, 1000),
		erClientset:           clientset,
	}, nil
}

func (c *Controller) registerEventHandlers() {
	c.eventInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.processEvent(obj.(*erapi.Event))
		},
		UpdateFunc: func(old, new interface{}) {
			c.processEvent(new.(*erapi.Event))
		},
	})
}

// Run will start the controller.
func (c *Controller) Run(stopChan <-chan struct{}) {
	go c.informerFactory.Start(stopChan)
	c.informerFactory.WaitForCacheSync(stopChan)
	c.registerEventHandlers()
	fmt.Println("Started")
	<-stopChan
}
