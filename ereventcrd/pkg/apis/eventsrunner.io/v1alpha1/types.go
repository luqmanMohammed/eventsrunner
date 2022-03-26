package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"


// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EREvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EREventSpec   `json:"spec"`
	Status EREventStatus `json:"status"`
}

type EREventSpec struct {
	EventType string `json:"eventType"`
}

type EREventStatus struct {
	Status string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EREventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []EREvent `json:"items"`
}
