package helpers

import (
	"context"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/index"
	"github.com/luqmanMohammed/eventsrunner/controller/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ruleHelper struct {
	client      client.Client
	listOptions []client.ListOption
}

// GetRuleOptions returns the option for the given ruleID if available
// else the default options will be returned
func (rh ruleHelper) GetRuleOptions(ctx context.Context, ruleID string) (*eventsrunneriov1alpha1.RulesOptsSpec, error) {
	var rulesOptsList eventsrunneriov1alpha1.RulesOptsList
	rh.listOptions = append(rh.listOptions, client.MatchingFields{index.RulesOptsRulesIndex: ruleID})
	if err := rh.client.List(ctx,
		&rulesOptsList,
		rh.listOptions...,
	); err != nil {
		return nil, err
	}
	if len(rulesOptsList.Items) == 0 {
		return &eventsrunneriov1alpha1.RulesOptsSpec{
			MaintainExecutionOrder:      utils.GetPointerFor[bool](true),
			RetainSuccessfulEventsCount: utils.GetPointerFor[int](5),
		}, nil
	}
	foundOpts := &rulesOptsList.Items[0].Spec
	if foundOpts.MaintainExecutionOrder == nil {
		foundOpts.MaintainExecutionOrder = utils.GetPointerFor[bool](true)
	}
	if foundOpts.RetainSuccessfulEventsCount == nil {
		foundOpts.RetainSuccessfulEventsCount = utils.GetPointerFor[int](5)
	}
	return foundOpts, nil
}
