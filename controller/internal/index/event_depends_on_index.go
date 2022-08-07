package index

import (
	"context"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlman "sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// EventDependsOnIndex is the name of the index on Events objects in cache for
	// faster lookup of Events by DependsOn field in their Spec
	EventDependsOnIndex string = "event_depends_on"
)

// registerEventDependsOnIndex registers the index on Events objects in cache for
// faster lookup of Events by DependsOn field in their Spec
func registerEventDependsOnIndex(ctx context.Context, mgr ctrlman.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&eventsrunneriov1alpha1.Event{},
		EventDependsOnIndex,
		func(o client.Object) []string {
			event := o.(*eventsrunneriov1alpha1.Event)
			if event.Spec.DependsOn == "" {
				return []string{}
			}
			return []string{event.Spec.DependsOn}
		},
	)
}
