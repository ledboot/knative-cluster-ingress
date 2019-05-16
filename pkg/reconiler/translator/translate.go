package translator

import (
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler/kong"
)

func translateProxy(namespace string) (*kong.KongState, error) {
	return &kong.KongState{}, nil
}
