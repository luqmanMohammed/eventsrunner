package handlers

import (
	"context"
	"sort"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/internal/index"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type eventHandler struct {
	client.Client
	controllerNamespace string
}

func filterEventsIfCreatedAfter(d []eventsrunneriov1alpha1.Event, s []eventsrunneriov1alpha1.Event, createdAfter *metav1.Time) {
	for _, obj := range s {
		if obj.CreationTimestamp.Before(createdAfter) {
			d = append(d, obj)
		}
	}
}

func (eh eventHandler) FindEventDependsOn(ctx context.Context, event *eventsrunneriov1alpha1.Event) (*eventsrunneriov1alpha1.Event, error) {
	var eventsList eventsrunneriov1alpha1.EventList
	if err := eh.List(
		ctx,
		&eventsList,
		client.MatchingFields{
			index.EventResourceIDIndex: event.Spec.ResourceID,
		},
		client.InNamespace(eh.controllerNamespace),
	); err != nil {
		return nil, err
	}
	if len(eventsList.Items) == 0 {
		return nil, nil
	}
	destEvents := make([]eventsrunneriov1alpha1.Event, 0, len(eventsList.Items))
	filterEventsIfCreatedAfter(
		destEvents,
		eventsList.Items,
		&event.CreationTimestamp,
	)
	if len(destEvents) == 1 {
		return &destEvents[0], nil
	}
	sort.SliceStable(
		destEvents,
		func(i, j int) bool {
			return destEvents[i].CreationTimestamp.Before(&destEvents[j].CreationTimestamp)
		},
	)
	return &destEvents[(len(destEvents) - 1):][0], nil
}
