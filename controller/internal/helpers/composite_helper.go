package helpers

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CompositeHelper includes includes handlers to handle events, runners and jobs
type CompositeHelper struct {
	runnerHelper
	eventsHelper
	ruleHelper
	jobHelper
}

// NewCompositeHelper returns a new composite handler
func NewCompositeHelper(controllerLabel, controllerNamespace string, k8sClient client.Client) CompositeHelper {
	listOptions := []client.ListOption{
		client.InNamespace(controllerNamespace),
		client.HasLabels{controllerLabel},
	}
	return CompositeHelper{
		runnerHelper: runnerHelper{
			client:              k8sClient,
			listOptions:         listOptions,
			controllerNamespace: controllerNamespace,
		},
		eventsHelper: eventsHelper{
			client:      k8sClient,
			listOptions: listOptions,
		},
		ruleHelper: ruleHelper{
			client:      k8sClient,
			listOptions: listOptions,
		},
		jobHelper: jobHelper{
			client:      k8sClient,
			listOptions: listOptions,
		},
	}

}
