package index

import (
	"context"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlman "sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// EventResourceIDIndex is the name of the index on Events objects in cache for
	// faster lookup of Events by ResourceId field in their Spec
	EventResourceIDIndex string = "event_resource_id"
)

// registerEventResourceIDIndex registers the index on Events objects in cache for
// faster lookup of Events by DependsOn field in their Spec
func registerEventResourceIDIndex(ctx context.Context, mgr ctrlman.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		ctx,
		&eventsrunneriov1alpha1.Event{},
		EventResourceIDIndex,
		func(o client.Object) []string {
			event := o.(*eventsrunneriov1alpha1.Event)
			if event.Spec.ResourceID == "" {
				return []string{}
			}
			return []string{event.Spec.ResourceID}
		},
	)
}
