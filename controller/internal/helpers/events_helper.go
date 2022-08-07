package helpers

import (
	"context"
	"sort"

	logr "github.com/go-logr/logr"
	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/index"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// eventsHelper contains helper functions to handle events related operations.
// These functions are exposed via the CompositeHelper.
type eventsHelper struct {
	helperLog           logr.Logger
	client              client.Client
	controllerNamespace string
}

// filterEventsIfCreatedAfter filters events if created after the given timestamp.
func filterEventsIfCreatedAfter(d []eventsrunneriov1alpha1.Event, s []eventsrunneriov1alpha1.Event, createdAfter *metav1.Time) {
	for _, obj := range s {
		if obj.CreationTimestamp.Before(createdAfter) {
			d = append(d, obj)
		}
	}
}

/*	FindEventDependsOn returns the event that the specified event depends on.
	Event Dependecy is maintained to make sure the jobs are run in the order of the events created.
	Events created for a specific resource will belong to a dependency group and events with the same
	dependency group will be run in the order of the events created.

	Lastetly created event will depend on the event that was created before it in the same dependency group.
	To set the dependency, set the `dependsOn` field of the event.
*/
func (eh eventsHelper) FindEventDependsOn(ctx context.Context, event *eventsrunneriov1alpha1.Event) (*eventsrunneriov1alpha1.Event, error) {
	var eventsList eventsrunneriov1alpha1.EventList
	if err := eh.client.List(
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

// UpdateEventStatus updates the status of the event.
func (eh eventsHelper) UpdateEventStatus(ctx context.Context, event *eventsrunneriov1alpha1.Event, state eventsrunneriov1alpha1.EventState, message string) {
	event.Status.State = state
	event.Status.Message = message
	if err := eh.client.Update(ctx, event); err != nil {
		eh.helperLog.V(1).Error(err, "failed to update event status")
	}
}