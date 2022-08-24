package helpers

import (
	"context"
	"fmt"
	"sort"

	logr "github.com/go-logr/logr"
	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/index"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// eventsHelper contains helper functions to handle events related operations.
// These functions are exposed via the CompositeHelper.
type eventsHelper struct {
	helperLog   logr.Logger
	client      client.Client
	listOptions []client.ListOption
}

// filterEvents filters events if created after the given timestamp.
func filterEvents(s []eventsrunneriov1alpha1.Event, checkEvent *eventsrunneriov1alpha1.Event) []eventsrunneriov1alpha1.Event {
	d := make([]eventsrunneriov1alpha1.Event, 0, len(s))
	for _, obj := range s {
		if checkEvent.Namespace == obj.Namespace && checkEvent.Name == obj.Name {
			continue
		}
		if obj.CreationTimestamp.Before(&checkEvent.CreationTimestamp) || obj.CreationTimestamp.Equal(&checkEvent.CreationTimestamp) {
			d = append(d, obj)
		}
	}
	return d
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
	eh.listOptions = append(
		eh.listOptions,
		client.MatchingFields{index.EventDependsOnIndex: event.Name})
	if err := eh.client.List(
		ctx,
		&eventsList,
		eh.listOptions...,
	); err != nil {
		return nil, err
	}
	if len(eventsList.Items) == 0 {
		return nil, nil
	}
	fmt.Printf("%+v\n", eventsList.Items)
	destEvents := filterEvents(eventsList.Items, event)
	if len(destEvents) == 1 {
		return &destEvents[0], nil
	}
	sort.SliceStable(
		destEvents,
		func(i, j int) bool {
			return destEvents[i].CreationTimestamp.Before(&destEvents[j].CreationTimestamp)
		},
	)
	fmt.Printf("%+v\n", destEvents)
	return &destEvents[(len(destEvents) - 1):][0], nil
}

// StillDependent returns true if the event is still dependent on other events.
func (eh eventsHelper) StillDependent(ctx context.Context, eventName string) (bool, error) {
	if err := eh.client.Get(ctx, client.ObjectKey{Name: eventName}, &eventsrunneriov1alpha1.Event{}); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UpdateEventStatus updates the status of the event.
func (eh eventsHelper) UpdateEventStatus(ctx context.Context, event *eventsrunneriov1alpha1.Event, state eventsrunneriov1alpha1.EventState, message string) error {
	if event.Status.State == state && event.Status.Message == message {
		return nil
	}
	event.Status.State = state
	event.Status.Message = message
	if err := eh.client.Update(ctx, event); err != nil {
		return err
	}
	return nil
}
