package handlers

import "sigs.k8s.io/controller-runtime/pkg/client"

// CompositeHandler includes includes handlers to handle events, runners and jobs
type CompositeHandler struct {
	RunnerManager
}

// NewCompositeHandler returns a new composite handler
func NewCompositeHandler(controllerName, controllerNamespace string, client client.Client) CompositeHandler {
	return CompositeHandler{
		RunnerManager: RunnerManager{
			Client:                client,
			RunnerNamespace:       controllerNamespace,
			RunnerIdentifierLabel: controllerNamespace,
		},
	}
}
