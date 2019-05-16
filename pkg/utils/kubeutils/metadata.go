package kubeutils

import (
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FromKubeMeta(meta metav1.ObjectMeta) reconiler.Metadata {
	return reconiler.Metadata{
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
