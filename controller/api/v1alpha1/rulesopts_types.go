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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RulesOptsSpec defines the desired state of RulesOpts
type RulesOptsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of RulesOpts. Edit rulesopts_types.go to remove/update
	// +optional
	Concurrency *int `json:"concurrency,omitempty"`
	// +optional
	MaintainExecutionOrder *bool `json:"maintainExecutionOrder,omitempty"`
	// +optional
	RetryCount *int `json:"retryCount,omitempty"`
	// +optional
	RetryInterval *metav1.Time `json:"retryInterval,omitempty"`
	// +optional
	RetainFailedEventsCount *int `json:"retainFailedEventsCount,omitempty"`
	// +optional
	RetainSuccessfulEventsCount *int `json:"retainSuccessfulEventsCount,omitempty"`
}

//+kubebuilder:object:root=true

// RulesOpts is the Schema for the rulesopts API
type RulesOpts struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Rules []string      `json:"rules,omitempty"`
	Spec  RulesOptsSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// RulesOptsList contains a list of RulesOpts
type RulesOptsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RulesOpts `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RulesOpts{}, &RulesOptsList{})
}
