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
	// ADDED is the type of the event when a new resource is added
	ADDED EventType = "added"
	// UPDATED is the type of the event when a resource is updated
	UPDATED EventType = "updated"
	// DELETED is the type of the event when a resource is deleted
	DELETED EventType = "deleted"
)

// EventSpec will be used to store the exact request recieved by the eventsrunner-api
type EventSpec struct {
	ResourceID string    `json:"resourceID"`
	RuleID     string    `json:"ruleID"`
	EventType  EventType `json:"eventType"`
	// +optional
	EventData []string `json:"eventData"`
}

// EventState Depicts the current state of the event
type EventState string

const (
	// PENDING is the state of the event when is is ready to be processed
	PENDING EventState = "pending"
	// PROCESSING is the state of the event when it is being processed
	PROCESSING EventState = "processing"
	// PROCESSED is the state of the event when it is processed
	PROCESSED EventState = "processed"
)

// EventStatus will be used to store the current status of the event
type EventStatus struct {
	State EventState `json:"status" default:"pending"`
	// +optional
	Retries int `json:"retries" default:"0"`
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

// RunnerBinding binds a runner to one or more rules stating this runner
// should picked for this rule or rules.
// Runner level configuration can be overidden here, but it is recommended to
// keep it to minimal.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RunnerBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Runner            string `json:"runner"`
	// +optional
	Overides v1.PodSpec `json:"overides"`
	Rules    []string   `json:"rules"`
}

// RunnerBindingList is a list of RunnerBindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RunnerBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunnerBinding `json:"items"`
}
