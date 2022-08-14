package helpers

import "sigs.k8s.io/controller-runtime/pkg/client"

type jobHelper struct {
	client      client.Client
	listOptions []client.ListOption
}
