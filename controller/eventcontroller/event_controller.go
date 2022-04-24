package eventcontroller

import erinformer "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/eventsrunner.io/v1alpha1"

// EventController handles events
type EventController struct {
	eventNamespace string
	eventInformer  erinformer.EventInformer
}
