/*
Copyright © 2021 Luqman Mohammed m.luqman077@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"fmt"
	"time"

	"github.com/luqmanMohammed/eventsrunner/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	erapi "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	erclient "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned"
	erinformer "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions"
	"github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers/externalversions/internalinterfaces"
)

func main() {
	config, err := common.GetKubeAPIConfig("")
	if err != nil {
		panic(err)
	}
	client, err := erclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	sharedInformer := erinformer.NewFilteredSharedInformerFactory(client, 0, "default", internalinterfaces.TweakListOptionsFunc(func(lo *metav1.ListOptions) {
	}))

	sharedInformer.Eventsrunner().V1alpha1().Events().Informer().AddIndexers(cache.Indexers{
		"event-id": func(obj interface{}) ([]string, error) {
			event := obj.(*erapi.Event)
			return []string{fmt.Sprintf("%s-%s-%s", event.Spec.EventType, event.Spec.ResourceID, event.Spec.RuleID)}, nil
		},
	})

	stopChan := make(chan struct{})

	go func() {
		sharedInformer.Start(stopChan)
	}()

	for range time.NewTicker(time.Second * 1).C {
		itemList, err := sharedInformer.Eventsrunner().V1alpha1().Events().Informer().GetIndexer().ByIndex("event-id", "added-test-resource-test-rule")
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, item := range itemList {
			fmt.Println(item)
		}
	}
	<-stopChan

	// sharedInformer.Eventsrunner().V1alpha1().Events().Informer().AddEventHandler(
	// 	cache.ResourceEventHandlerFuncs{
	// 		AddFunc:    func(obj interface{}) {},
	// 		UpdateFunc: func(oldObj, newObj interface{}) {},
	// 		DeleteFunc: func(obj interface{}) {},
	// 	},
	// )

	//cmd.Execute()
}
