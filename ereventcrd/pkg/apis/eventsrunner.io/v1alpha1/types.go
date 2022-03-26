package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// EREvent depicts an event that is processed/to be processed by the events runner.
// EREvents will be requests recieved by the eventsrunner-api, which will be stored as
// CRDs for persistance. Stored EREvents will be processed by the eventsrunner-controller
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EREvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EREventSpec `json:"spec"`
	// +optional
	Status EREventStatus `json:"status"`
}

// EREventType define the type of the event
type EREventType string

const (
	// ADDED is the type of the event when a new resource is added
	ADDED EREventType = "added"
	// UPDATED is the type of the event when a resource is updated
	UPDATED EREventType = "updated"
	// DELETED is the type of the event when a resource is deleted
	DELETED EREventType = "deleted"
)

// EREventSpec will be used to store the exact request recieved by the eventsrunner-api
type EREventSpec struct {
	ResourceID string      `json:"resourceID"`
	RuleID     string      `json:"ruleID"`
	EventType  EREventType `json:"eventType"`
	// +optional
	EventData []map[string]string `json:"eventData"`
}

// EREventState Depicts the current state of the event
type EREventState string

const (
	// PENDING is the state of the event when is is ready to be processed
	PENDING EREventState = "pending"
	// PROCESSING is the state of the event when it is being processed
	PROCESSING EREventState = "processing"
	// PROCESSED is the state of the event when it is processed
	PROCESSED EREventState = "processed"
)

// EREventStatus will be used to store the current status of the event
type EREventStatus struct {
	State   EREventState `json:"status" default:"pending"`
	// +optional
	Retries int          `json:"retries" default:"0"`
}

// EREventList is a list of EREvents
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EREventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []EREvent `json:"items"`
}
