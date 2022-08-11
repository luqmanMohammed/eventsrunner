package index

import (
	"context"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlman "sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// RulesOptsRulesIndex is the name of the index on RulesOpts objects in cache for
	// faster lookup of RulesOpts by Rules field in their Spec
	RulesOptsRulesIndex string = "rulesopts_rules"
)

// registerRulesOptsRulesIndex registers the index on RulesOpts objects in cache for
// faster lookup of RulesOpts by Rules field in their Spec
func registerRulesOptsRulesIndex(ctx context.Context, mgr ctrlman.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&eventsrunneriov1alpha1.RulesOpts{},
		RulesOptsRulesIndex,
		func(o client.Object) []string {
			rulesOpts := o.(*eventsrunneriov1alpha1.RulesOpts)
			return rulesOpts.Rules
		},
	)
}
