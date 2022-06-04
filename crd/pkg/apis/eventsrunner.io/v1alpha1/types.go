package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Event depicts an event that is processed/to be processed by the events runner.
// EREvents will be requests recieved by the eventsrunner-api, which will be stored as
// CRDs for persistance. Stored EREvents will be processed by the eventsrunner-controller
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EventSpec `json:"spec"`
	// +optional
	Status EventStatus `json:"status"`
}

// EventType define the type of the event
type EventType string

const (
	// EventTypeAdded is the type of the event when a new resource is added
	EventTypeAdded EventType = "added"
	// EventTypeUpdated is the type of the event when a resource is updated
	EventTypeUpdated EventType = "updated"
	// EventTypeDeleted is the type of the event when a resource is deleted
	EventTypeDeleted EventType = "deleted"
)

// EventSpec will be used to store the exact request recieved by the eventsrunner-api
type EventSpec struct {

	// EventOriginID will be used to uniquely identify the event origin.
	// If empty " " will be used. Combination of EventOriginID and ResourceID should be unique
	// for each resource and will be used in the following combination to create the event
	// <EventOriginID>-<ResourceID>-<Genrated Random>
	// +optional
	EventOriginID string `json:"eventOriginID"`
	// ResourceID along with the event origin id will be used to uniquely identify the resource
	// This is required to prevent concurrent changes to the resource.
	ResourceID string `json:"resourceID"`
	// RuleID is the id of the rule that is being applied to the resource. Rule will be used to
	// determine the runner to be used to process the event.
	RuleID string `json:"ruleID"`
	// EventType is the type of the change on the resource. Can be added, updated or deleted
	// Used to configure a specific runner for a specific type of change.
	EventType EventType `json:"eventType"`
	// EventData contains any extra data that is passed to the runner.
	// +optional
	EventData []string `json:"eventData"`
}

// EventState Depicts the current state of the event
type EventState string

const (
	// EventStateEmpty is the initial state of the event
	EventStateEmpty EventState = ""
	// EventStatePending is the state of the event is yet to be assigned a runner
	EventStatePending EventState = "pending"
	// EventStateWaiting is the state of the event where it is waiting for a dependent
	// event to be processed before it can be processed
	EventStateWaiting EventState = "waiting"
	// EventStateAssigned is the state of the event is assigned to a runner
	EventStateAssigned EventState = "assigned"
	// EventStateProcessing is the state of the event is being processed by the runner
	EventStateProcessing EventState = "processing"
	// EventStateCompleted is the state of the event is processed by the runner successfully
	EventStateCompleted EventState = "completed"
	// EventStateFailed is the state of the event is processed by the runner failed
	// will be retried if retries are available
	EventStateFailed EventState = "failed"
	// EventStateRetriesExpired is the state of the event is processed by the runner failed for
	// the number of retries available
	EventStateRetriesExpired EventState = "retriesExpired"
	// EventStateDependentFailed is the state of the event when the dependent event failed
	EventStateDependentFailed EventState = "dependentFailed"
)

// EventStatus will be used to store the current status of the event
type EventStatus struct {
	// State depicts the current state of the event
	// +optional
	State EventState `json:"status" default:"pending"`
	// Retries depicts the number of retries that have been done for the event
	// +optional
	Retries int `json:"retries" default:"0"`
	// RuleBindingName depicts the name of the rule binding that is
	// associated with the event
	// +optional
	RuleBindingName string `json:"ruleBindingName"`
	// RunnerName depicts the name of the runner that is associated with the event
	// +optional
	RunnerName string `json:"runnerName"`
	// DependentEventID depicts the id of the dependent event that is associated with the event
	// +optional
	DependentEventID string `json:"dependentEventID"`
}

// EventList is a list of EREvents
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Event `json:"items"`
}

// Runner defines the spec to create a new task runner (K8s Job) which will be used
// run tasks on events.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v1.PodSpec `json:"spec"`
}

// RunnerList is a list of task runners
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

// RunnerOveride defines overrides for the runner spec at the binding level.
// This allows use of a single runner for multiple bindings with slight
// variations in the spec.
type RunnerOveride struct {
	// Override the service account in the runner spec
	// +optional
	ServiceAccount string `json:"serviceAccount"`
	// Override container spec in the runner spec.
	// Map of container name to container overide spec
	// +optional
	Containers map[string]RunnerContainerOveride `json:"containers"`
}

// RunnerContainerOveride defines overrides for the runner container in the
// runner spec.
type RunnerContainerOveride struct {
	// Override the environment variables in the container spec
	// If the env name matches the value will be replaced
	// +optional
	Env []v1.EnvVar `json:"env,omitempty"`
	// Override the command in the container spec
	// +optional
	Command []string `json:"command,omitempty"`
	// Override the args in the container spec
	// +optional
	Args []string `json:"args,omitempty"`
	// Override the volume mounts in the container spec
	// +optional
	Image string `json:"image,omitempty"`
}

// RunnerBinding binds a runner to one or more rules stating this runner
// should picked for this rule or rules.
// Runner level configuration can be overidden here, but it is recommended to
// keep it to minimal.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RunnerBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Name of the runner which is to be picked for this binding
	Runner string `json:"runner"`
	// Overrides for the runner spec at the binding level
	// +optional
	Overides *RunnerOveride `json:"overides"`
	// Rules that this binding is associated with.
	// Rules must be associated with a single binding, if multiple bindings
	// are found the first one will be picked which can be unpredictable.
	// Rules can associated as follows:
	// 1. - rule1 - Runner to be picked for event types for this rule
	// 2. - added:rule1 - Runner to be picked for added event type of this rule
	// +optional
	Rules []string `json:"rules"`
}

// RunnerBindingList is a list of RunnerBindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RunnerBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunnerBinding `json:"items"`
}
