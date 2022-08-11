package helpers

import (
	"context"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/index"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ruleHelper struct {
	client          client.Client
	runnerNamespace string
	controllerLabel string
}

func getPointerOf[T any](v T) *T {
	return &v
}

func (rh ruleHelper) GetRuleOptions(ctx context.Context, ruleID string) (*eventsrunneriov1alpha1.RulesOptsSpec, error) {
	var rulesOptsList eventsrunneriov1alpha1.RulesOptsList
	if err := rh.client.List(ctx,
		&rulesOptsList,
		client.InNamespace(rh.runnerNamespace),
		client.HasLabels{rh.controllerLabel},
		client.MatchingFields{
			index.RulesOptsRulesIndex: ruleID,
		}); err != nil {
		return nil, err
	}
	if len(rulesOptsList.Items) == 0 {
		return &eventsrunneriov1alpha1.RulesOptsSpec{
			MaintainExecutionOrder:      getPointerOf(true),
			RetainSuccessfulEventsCount: getPointerOf(5),
		}, nil
	}
	return &rulesOptsList.Items[0].Spec, nil
}
