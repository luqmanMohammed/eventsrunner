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

// +kubebuilder:validation:Required

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AddedEventType is the type of event when a resource is added.
	AddedEventType EventType = "added"
	// ModifiedEventType is the type of event when a resource is updated.
	ModifiedEventType EventType = "modified"
	// DeletedEventType is the type of event when a resource is deleted.
	DeletedEventType EventType = "deleted"
)

// +kubebuilder:validation:Enum={added,modified,deleted}

// EventType is the type of event
type EventType string

// EventSpec defines the desired state of Event
type EventSpec struct {
	RuleID     string    `json:"ruleID,omitempty"`
	ResourceID string    `json:"resourceID,omitempty"`
	EventType  EventType `json:"eventType,omitempty"`
	// +kubebuilder:validation:Schemaless
	// +optional
	EventData map[string]interface{} `json:"eventData,omitempty"`
	// +optional
	RunnerName string `json:"runnerName,omitempty"`
}

// EventStatus defines the observed state of Event
type EventStatus struct {
	// +optional
	JobName string `json:"jobName,omitempty"`
	// +optional
	DependsOn string `json:"dependsOn,omitempty"`
	// +optional
	Completed bool `json:"completed,omitempty"`
	// +optional
	Retries int `json:"retries,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Event is the Schema for the events API
type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSpec   `json:"spec,omitempty"`
	Status EventStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventList contains a list of Event
type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Event `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Event{}, &EventList{})
}
