/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*

New Event
-> Update RunnerName if not set
-> Check if has any depen
-> If no dependencies, shedule a job
-> If dependencies, wait for them to be completed
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/helpers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventReconciler reconciles a Event object
type EventReconciler struct {
	client.Client
	Scheme                      *runtime.Scheme
	CompositeHelper             helpers.CompositeHelper
	ControllerNamespace         string
	ControllerLabelKey          string
	ControllerLabelValue        string
	RequeueAfterSysFailDuration time.Duration
}

//+kubebuilder:rbac:groups=eventsrunner.io,resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eventsrunner.io,resources=events/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eventsrunner.io,resources=events/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *EventReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// TODO: Configure retry mechanisum for failed states if required
	logger := log.FromContext(ctx)

	// Get the Event object from the request.
	// If the Event object does not exist, assume that is was deleted and ignore.
	var event eventsrunneriov1alpha1.Event
	if err := r.Get(ctx, req.NamespacedName, &event); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.V(1).Error(err, "Failed to get error during reconciliation")
		return ctrl.Result{}, nil
	}

	updateEventFailedStatus := func(err error, msg string) error {
		return r.CompositeHelper.UpdateEventStatus(
			ctx,
			&event,
			eventsrunneriov1alpha1.EventStateFailed,
			fmt.Sprintf("ERROR: %v : %s", err, msg),
		)
	}

	// Check if runner name is set, if not, resolve runner and set it
	if event.Spec.RunnerName == "" {
		runnerName, err := r.CompositeHelper.ResolveRunner(ctx, &event)
		if err != nil {
			logger.V(1).Error(err, "Failed to resolve runner")
			return ctrl.Result{}, updateEventFailedStatus(err, "Failed to resolve runner")
		}
		event.Spec.RunnerName = runnerName
		if err := r.Update(ctx, &event); err != nil {
			logger.V(1).Error(err, "Failed to update runner name")
			return ctrl.Result{}, updateEventFailedStatus(err, "Failed to update runner name")
		}
		return ctrl.Result{}, nil
	}

	// Check if the job is already created
	// Job updates will be trigger this same reconciler via owner reference
	var job batchv1.Job
	if err := r.Get(ctx, client.ObjectKey{Namespace: event.Namespace, Name: event.Name}, &job); err != nil {
		if !errors.IsNotFound(err) {
			logger.V(1).Error(err, "Failed to get job")
			return ctrl.Result{}, r.CompositeHelper.UpdateEventStatus(
				ctx,
				&event,
				eventsrunneriov1alpha1.EventStateFailed,
				fmt.Sprintf("ERROR %v : Failed to get job", err),
			)
		}

		// RuleOptions objects are used to store options associated with a rule
		// Checking if the controller should maintain event dependency
		ruleOptions, err := r.CompositeHelper.GetRuleOptions(
			ctx,
			event.Spec.RuleID,
		)
		if err != nil {
			logger.V(2).Error(err, "Failed to resolve rule options")
			return ctrl.Result{}, err
		}

		if !(*ruleOptions.MaintainExecutionOrder) || event.Spec.DependsOn == "" {
			if *ruleOptions.MaintainExecutionOrder {
				dependsOnEvent, err := r.CompositeHelper.FindEventDependsOn(ctx, &event)
				if err != nil {
					logger.V(1).Error(err, "Failed to find depends on")
					return ctrl.Result{}, r.CompositeHelper.UpdateEventStatus(
						ctx,
						&event,
						eventsrunneriov1alpha1.EventStateFailed,
						fmt.Sprintf("ERROR %v : Failed to find depends on", err),
					)
				}
				if dependsOnEvent != nil {
					event.Spec.DependsOn = dependsOnEvent.Name
					if err := r.Update(ctx, &event); err != nil {
						logger.V(1).Error(err, "Failed to update depends on")
						return ctrl.Result{}, r.CompositeHelper.UpdateEventStatus(
							ctx,
							&event,
							eventsrunneriov1alpha1.EventStateFailed,
							fmt.Sprintf("ERROR %v : Failed to update depends on", err),
						)
					}
					return ctrl.Result{}, nil
				}
			}
			// Shedule
			runner, err := r.CompositeHelper.GetRunner(ctx, event.Spec.RunnerName)
			if err != nil {
				logger.V(1).Error(err, "Failed to get runner")
				return ctrl.Result{}, r.CompositeHelper.UpdateEventStatus(
					ctx,
					&event,
					eventsrunneriov1alpha1.EventStateFailed,
					fmt.Sprintf("ERROR %v : Failed to get runner", err),
				)
			}
			// TODO: Move this under job helper : Prepare Job
			createJob := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      event.Name,
					Namespace: event.Namespace,
					Labels: map[string]string{
						"eventsrunner.io/resourceId": event.Spec.ResourceID,
						"eventsrunner.io/eventType":  string(event.Spec.EventType),
						r.ControllerLabelKey:         r.ControllerLabelValue,
					},
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec(runner.Spec),
				},
			}
			if err := controllerutil.SetControllerReference(&event, &createJob, r.Scheme); err != nil {
				logger.Error(err, "Unable to set controller reference")
				return ctrl.Result{}, r.CompositeHelper.UpdateEventStatus(
					ctx,
					&event,
					eventsrunneriov1alpha1.EventStateFailed,
					fmt.Sprintf("ERROR %v : Unable to set controller reference", err),
				)
			}
			if err := r.Create(ctx, &createJob); err != nil {
				logger.Error(err, "Unable to create job")
				return ctrl.Result{}, r.CompositeHelper.UpdateEventStatus(
					ctx,
					&event,
					eventsrunneriov1alpha1.EventStateFailed,
					fmt.Sprintf("ERROR %v : Unable to create job", err),
				)
			}
		} else {
			logger.V(2).Info(fmt.Sprintf("Event is depending on %s", event.Spec.DependsOn))
			return ctrl.Result{}, nil
		}
	} else {
		// If the job is already sheduled check if it has succeeded or failed
		switch r.CompositeHelper.GetJobStatus(job) {
		case batchv1.JobComplete:
			// TODO: Check if any events depends on this job and set it to empty
			// TODO: Only delete if outside keep success limit
			return ctrl.Result{}, r.Delete(ctx, &event)
		case batchv1.JobFailed:
			// TODO: Delete event if outside keep failed limit
			// TODO: Update status event
			return ctrl.Result{}, nil
		case batchv1.JobSuspended:
			// Ideally controller managed Job should never reach this state
			// Delete if reached?
			return ctrl.Result{}, r.Delete(ctx, &event)
		}
	}
	return ctrl.Result{}, nil
}

// EventFilter is a predicate that filters events that are not of interest to
// the controller. Controller will only process events which are in the
// controller's namespace and have the controller label.
// Return true if event is of interest to the controller.
func (r *EventReconciler) EventFilter(obj client.Object) bool {
	if obj.GetNamespace() != r.ControllerNamespace {
		return false
	}
	labelValue, ok := obj.GetLabels()[r.ControllerLabelKey]
	if !ok {
		return false
	}
	if labelValue != r.ControllerLabelValue {
		return false
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventsrunneriov1alpha1.Event{}).
		Owns(&batchv1.Job{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return r.EventFilter(e.Object)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return r.EventFilter(e.ObjectNew)
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return r.EventFilter(e.Object)
			},
		}).
		Complete(r)
}
