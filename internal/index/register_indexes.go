package index

import (
	"context"

	ctrlman "sigs.k8s.io/controller-runtime/pkg/manager"
)

// RegisterIndexes registers the indexes on the objects in cache for faster lookup of objects
// by fields in their Spec or metadata.
// TODO: Handler logger
func RegisterIndexes(ctx context.Context, mgr ctrlman.Manager) error {
	if err := registerEventResourceIDIndex(ctx, mgr); err != nil {
		return err
	}
	if err := registerEventDependsOnIndex(ctx, mgr); err != nil {
		return err
	}
	if err := registerRunnerBindingRulesIDIndex(ctx, mgr); err != nil {
		return err
	}
	return nil
}
