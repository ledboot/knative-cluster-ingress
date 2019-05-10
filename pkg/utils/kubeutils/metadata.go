package kubeutils

import (
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FromKubeMeta(meta metav1.ObjectMeta) v1.Metadata {
	return v1.Metadata{
		Name:            meta.Name,
		Namespace:       meta.Namespace,
		ResourceVersion: meta.ResourceVersion,
		Labels:          copyMap(meta.Labels),
		Annotations:     copyMap(meta.Annotations),
	}
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	res := map[string]string{}
	for k, v := range m {
		res[k] = v
	}
	return res
}
