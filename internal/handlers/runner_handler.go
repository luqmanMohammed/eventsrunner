package handlers

import (
	"context"
	"errors"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/internal/index"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RunnerManager is responsible for selecting correct runners for a given event
type RunnerManager struct {
	client.Client
	RunnerNamespace       string
	RunnerIdentifierLabel string
}

var (
	// ErrNoRunnerBindingFound is returned when no runner binding is found for rule
	ErrNoRunnerBindingFound error = errors.New("no runner binding found")
)

// ResolveRunner returns the runner for the given event
func (m RunnerManager) ResolveRunner(ctx context.Context, event *eventsrunneriov1alpha1.Event) (string, error) {
	// Get the runner binding for the event
	logger := log.FromContext(ctx)
	var runnerBindingList eventsrunneriov1alpha1.RunnerBindingList
	if err := m.List(ctx, &runnerBindingList, client.MatchingFields{index.RunnerBindingRulesIDIndex: event.Spec.RuleID}, client.InNamespace(m.RunnerNamespace), client.HasLabels{
		m.RunnerIdentifierLabel,
	}); err != nil {
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
func (m RunnerManager) GetRunner(ctx context.Context, runnerName string) (*eventsrunneriov1alpha1.Runner, error) {
	var runner eventsrunneriov1alpha1.Runner
	if err := m.Get(ctx, client.ObjectKey{Name: runnerName, Namespace: m.RunnerNamespace}, &runner); err != nil {
		return nil, err
	}
	return &runner, nil
}
