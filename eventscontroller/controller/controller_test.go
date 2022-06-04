package controller

import (
	"testing"

	common "github.com/luqmanMohammed/eventsrunner/common/pkg"
)

func TestController(t *testing.T) {
	kubeconfig := common.GetKubeAPIConfigOrDie("")
	controller, err := New(kubeconfig, "eventsrunner", "eventsrunner")
	if err != nil {
		t.Fatal(err)
	}
	stopChan := make(chan struct{})
	controller.Run(stopChan)
}
