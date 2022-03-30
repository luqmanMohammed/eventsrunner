package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName identifies the group name for the API
const GroupName = "eventsrunner.io"

// GroupVersion identifies the API version under the said group
const GroupVersion = "v1alpha1"

// SchemeGroupVersion uniquely identifies the eventsrunner.io API group and v1alpha1 API version
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	// SchemeBuilder is used to add custom schemes to the schema
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds the types of this group into the given scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Event{},
		&EventList{},
		&Runner{},
		&RunnerList{},
		&RunnerBinding{},
		&RunnerBindingList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func init() {
	SchemeBuilder.Register(addKnownTypes)
}
