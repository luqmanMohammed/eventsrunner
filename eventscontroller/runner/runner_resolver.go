package runner

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	common "github.com/luqmanMohammed/eventsrunner/common/pkg"
	erAPI "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
	erInformers "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/eventsrunner.io/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/internalinterfaces"
)

// runnerResolver is a resolves runners based on a event. Event type and Rule ID
// will be used to determine the specific binding to look for the runner. RunnerBinding
// CRDs are used to configure the runner for a set of rules.
type runnerResolver struct {
	namespace              string
	stopChan               chan struct{}
	runnerInformerFactory  erInformers.SharedInformerFactory
	runnerInformer         v1alpha1.RunnerInformer
	runnerBindingsInformer v1alpha1.RunnerBindingInformer
}

// newRunnerResolver creates a pointer to a new runner resolver.
// It registers the informer factory and creates informers runner and runner binding crds.
// CRDS in the confirgured namespace with the label "eventsrunner.io/controller=<controler_name>"
// will be considered.
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

	// custom index to find runner bindings by rule id
	runnerBindingInformer.Informer().AddIndexers(cache.Indexers{
		"rules": func(obj interface{}) ([]string, error) {
			binding := obj.(*erAPI.RunnerBinding)
			return binding.Rules, nil
		},
	})

	return &runnerResolver{
		namespace:              namespace,
		runnerInformerFactory:  inf,
		runnerInformer:         runnerInformer,
		runnerBindingsInformer: runnerBindingInformer,
	}, nil
}

// start starts the informer factory and waits for the informer to be synced.
// start is a blocking call, call stop to release.
func (r *runnerResolver) start() {
	r.stopChan = make(chan struct{})
	r.runnerInformerFactory.Start(r.stopChan)
	r.runnerInformerFactory.WaitForCacheSync(r.stopChan)
	<-r.stopChan
}

// stop stops the informer factory by closing the stop channel.
func (r *runnerResolver) stop() {
	close(r.stopChan)
}

// bindingNotFoundError is an error that is returned when no runner binding is
// found for a rule.
type bindingNotFoundError struct {
	ruleID string
}

func (e *bindingNotFoundError) Error() string {
	return fmt.Sprintf("No runner binding found for rule %s", e.ruleID)
}

// resolve resolves a runner based on the event and rule id taken from the event.
// If no runner binding is found for the rule id, an error is returned.
// If multiple runner bindings are found, the first one is used.
// If updateEvent is true, the event will be updated with the runner name and runner binding name.
func (r *runnerResolver) resolve(event *erAPI.Event, updateEvent bool) (*erAPI.Runner, *erAPI.RunnerBinding, error) {
	// get the runner binding for the rule id using the above initialized index
	bindingsIntList, err := r.runnerBindingsInformer.Informer().GetIndexer().ByIndex("rules", event.Spec.RuleID)
	if err != nil {
		return nil, nil, err
	}
	if len(bindingsIntList) == 0 {
		return nil, nil, &bindingNotFoundError{ruleID: event.Spec.RuleID}
	}
	binding := common.ConvertInterfaceSliceToTyped[*erAPI.RunnerBinding](bindingsIntList)[0]
	runnerName := binding.Runner
	runner, err := r.runnerInformer.Lister().Runners(r.namespace).Get(runnerName)
	if err != nil {
		return nil, nil, err
	}
	if updateEvent {
		event.Status.RunnerName = runnerName
		event.Status.RuleBindingName = binding.Name
	}
	return runner, binding, nil
}

// resolvePodSpec resolves a podSpec that can be used to create a runner pod based
// on the event provided.
// TODO: Re-evaluate this function. Merging pod specs can be genaralized.
func (r *runnerResolver) resolvePodSpec(event *erAPI.Event) (*v1.PodSpec, error) {
	runner, binding, err := r.resolve(event, true)
	// TODO: Move merge to a separate function
	mergedSpec := runner.Spec.DeepCopy()
	if err != nil {
		return nil, err
	}
	if binding.Overides == nil {
		return mergedSpec, nil
	}
	if binding.Overides.ServiceAccount != "" {
		mergedSpec.ServiceAccountName = binding.Overides.ServiceAccount
	}
	mergedContainers := make([]v1.Container, 0, len(mergedSpec.Containers))
	for _, container := range mergedSpec.Containers {
		overideContainer, ok := binding.Overides.Containers[container.Name]
		if ok {
			if overideContainer.Image != "" {
				container.Image = overideContainer.Image
			}
			if overideContainer.Command != nil {
				container.Command = overideContainer.Command
			}
			if overideContainer.Args != nil {
				container.Args = overideContainer.Args
			}
			mergedEnvs := make([]v1.EnvVar, 0, len(container.Env))
			for _, env := range container.Env {
				for _, overideEnv := range overideContainer.Env {
					if env.Name == overideEnv.Name {
						env.Value = overideEnv.Value
					}
				}
				mergedEnvs = append(mergedEnvs, env)
			}
			container.Env = mergedEnvs
		}
		mergedContainers = append(mergedContainers, container)
	}
	mergedSpec.Containers = mergedContainers
	return mergedSpec, nil
}
