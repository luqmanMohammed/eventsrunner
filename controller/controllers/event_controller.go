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
-> Check if has any depencies
-> If no dependencies, shedule a job
-> If dependencies, wait for them to be completed
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	eventsrunneriov1alpha1 "github.com/luqmanMohammed/eventsrunner/controller/api/v1alpha1"
	"github.com/luqmanMohammed/eventsrunner/controller/internal/helpers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventReconciler reconciles a Event object
type EventReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CompositeHelper helpers.CompositeHelper
}

func (r *EventReconciler) updateFailedEvent(ctx context.Context, logger logr.Logger, event *eventsrunneriov1alpha1.Event, message string) {
	event.Status.State = eventsrunneriov1alpha1.EventStateFailed
	event.Status.Message = message
	if err := r.Status().Update(ctx, event); err != nil {
		logger.Error(err, "unable to update event status")
	}
}

//+kubebuilder:rbac:groups=eventsrunner.io,resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eventsrunner.io,resources=events/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eventsrunner.io,resources=events/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *EventReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the Event object from the request.
	// If the Event object does not exist, assume that is was deleted and ignore.
	var event eventsrunneriov1alpha1.Event
	if err := r.Get(ctx, req.NamespacedName, &event); err != nil {
		logger.V(1).Info("unable to fetch Event")
		return ctrl.Result{}, nil
	}

	// Check if runner name is set, if not, set it
	if event.Spec.RunnerName == "" {
		logger.V(1).Info("Event has no runner name")
		runnerName, err := r.CompositeHelper.ResolveRunner(ctx, &event)
		if err != nil {
			logger.Error(err, "unable to resolve runner")
			r.updateFailedEvent(ctx, logger, &event, fmt.Sprintf("unable to resolve runner: %s", err.Error()))
			return ctrl.Result{}, err
		}
		event.Spec.RunnerName = runnerName
		if err := r.Update(ctx, &event); err != nil {
			logger.Error(err, "unable to update event")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Check if job is already created
	var job batchv1.Job
	if err := r.Get(ctx, client.ObjectKey{Namespace: event.Namespace, Name: event.Name}, &job); err != nil {
		if event.Spec.DependsOn == "" {
			runner, err := r.CompositeHelper.GetRunner(ctx, event.Spec.RunnerName)
			if err != nil {
				return ctrl.Result{}, err
			}
			createJob := batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      event.Name,
					Namespace: event.Namespace,
					Labels: map[string]string{
						"eventsrunner.io/resourceId": event.Spec.ResourceID,
						"eventsrunner.io/eventType":  string(event.Spec.EventType),
					},
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec(runner.Spec),
				},
			}
			if err := controllerutil.SetControllerReference(&event, &createJob, r.Scheme); err != nil {
				logger.Error(err, "unable to set controller reference")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, &createJob); err != nil {
				logger.Error(err, "unable to create job")
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, nil
		}
	} else {
		fmt.Println("Job already created", job.Name, job.Status.Succeeded)
		logger.V(1).Info("Job already created")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventsrunneriov1alpha1.Event{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
