package reconiler

import (
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
)

func LabelExistFilter(kind string) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if ing, ok := obj.(*v1alpha1.ClusterIngress); ok {
			return ing.TypeMeta.Kind == kind
		}
		return false
	}
}
