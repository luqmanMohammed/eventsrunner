package helpers

import (
	"context"
	"errors"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/index"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// runnerHelper is contains helper functions to handle runner objects and operations.
// Handles finding the runner for a given event and getting the runner for a given name.
// It is exposed via the CompositeHelper.
type runnerHelper struct {
	client              client.Client
	listOptions         []client.ListOption
	controllerNamespace string
}

var (
	// ErrNoRunnerBindingFound is returned when no runner binding is found for rule
	ErrNoRunnerBindingFound error = errors.New("no runner binding found")
)

// ResolveRunner returns the runner for the given event
func (rh runnerHelper) ResolveRunner(ctx context.Context, event *eventsrunneriov1alpha1.Event) (string, error) {
	// Get the runner binding for the event
	logger := log.FromContext(ctx)
	rh.listOptions = append(rh.listOptions, client.MatchingFields{index.RunnerBindingRulesIDIndex: event.Spec.RuleID})
	var runnerBindingList eventsrunneriov1alpha1.RunnerBindingList
	if err := rh.client.List(
		ctx, &runnerBindingList,
		rh.listOptions...,
	); err != nil {
		logger.Error(err, "Failed to list runner bindings")
		return "", err
	}
	if len(runnerBindingList.Items) == 0 {
		logger.Error(ErrNoRunnerBindingFound, "No runner binding found for event", "rule", event.Spec.RuleID)
		return "", ErrNoRunnerBindingFound
	}
	if len(runnerBindingList.Items) > 1 {
		logger.V(1).Info("Multiple runner bindings found for event", "rule", event.Spec.RuleID)
	}
	runnerName := runnerBindingList.Items[0].RunnerName
	return runnerName, nil
}

// GetRunner returns the runner for the given name
// TODO: Handle runner parameter overriding
func (rh runnerHelper) GetRunner(ctx context.Context, runnerName string) (*eventsrunneriov1alpha1.Runner, error) {
	var runner eventsrunneriov1alpha1.Runner
	if err := rh.client.Get(ctx, client.ObjectKey{Name: runnerName, Namespace: rh.controllerNamespace}, &runner); err != nil {
		return nil, err
	}
	return &runner, nil
}
