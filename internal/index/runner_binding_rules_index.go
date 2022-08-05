package index

import (
	"context"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/api/v1alpha1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlman "sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// RunnerBindingRulesIDIndex index is created on RunnerBinding objects in cache for
	// faster lookup of RunnerBinding by RuleID
	RunnerBindingRulesIDIndex string = "runner-binding-rules"
)

// registerRunnerBindingRulesIDIndex registers the index on RunnerBinding objects in cache for
// faster lookup of RunnerBinding by RuleID
func registerRunnerBindingRulesIDIndex(ctx context.Context, mgr ctrlman.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&eventsrunneriov1alpha1.RunnerBinding{},
		RunnerBindingRulesIDIndex,
		func(o client.Object) []string {
			return o.(*eventsrunneriov1alpha1.RunnerBinding).Rules
		},
	)
}
