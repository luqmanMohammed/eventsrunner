package helpers

import "sigs.k8s.io/controller-runtime/pkg/client"

// CompositeHelper includes includes handlers to handle events, runners and jobs
type CompositeHelper struct {
	runnerHelper
	eventHelper
}

// NewCompositeHelper returns a new composite handler
func NewCompositeHelper(controllerName, controllerNamespace string, client client.Client) CompositeHelper {
	return CompositeHelper{
		runnerHelper: runnerHelper{
			client:                client,
			runnerNamespace:       controllerNamespace,
			runnerIdentifierLabel: controllerNamespace,
		},
	}
}
