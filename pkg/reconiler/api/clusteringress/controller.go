package clusteringress

import (
	"encoding/json"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler/api/v1"
	"github.com/ledboot/knative-cluster-ingress/pkg/utils/kubeutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

type KnativeController struct {
	*reconiler.Base
}

func NewController(opts reconiler.Options) *KnativeController {
	c := &KnativeController{
		reconiler.NewBase(opts),
	}
	return c
}

func (c *KnativeController) Start() {
	go c.watchClusterIngress()
	//go watchPod(c)
}

func (c *KnativeController) watchClusterIngress() {
	w, err := c.KnativeClientSet.NetworkingV1alpha1().ClusterIngresses().Watch(metav1.ListOptions{})
	if err != nil {
		c.Logger.Fatal(err)
	}
	for {
		select {
		case event := <-w.ResultChan():
			switch event.Type {
			case watch.Error:
				c.Logger.Errorf("error during watch: %v", event)
			default:
				c.listClusterIngress()
			}

		}
	}
}

func (c *KnativeController) listClusterIngress() {
	ingressObjList, err := c.KnativeClientSet.NetworkingV1alpha1().ClusterIngresses().List(metav1.ListOptions{})
	if err != nil {
		c.Logger.Errorf("listing ingressObjs at cluster level get error : %s", err)
	}

	for _, ingressObj := range ingressObjList.Items {
		c.Logger.Info("ingress :", ingressObj)
		//for i, r := range ingressObj.Spec.Rules {
		//	host := r.Hosts
		//	serviceName :=
		//}
		resource, err := fromKube(&ingressObj)
		if err != nil {
			c.Logger.Errorf("format ingerss error : %s", err)
		}
		c.Logger.Infof("get ingress :", resource)
	}

}

func fromKube(ingress *v1alpha1.ClusterIngress) (*v1.ClusterIngress, error) {
	rawSpec, err := json.Marshal(ingress.Spec)
	if err != nil {
		return nil, err
	}
	spec := &v1.Any{
		TypeUrl: v1.TypeUrl,
		Value:   rawSpec,
	}

	rawStatus, err := json.Marshal(ingress.Status)
	if err != nil {
		return nil, err
	}

	status := &v1.Any{
		TypeUrl: v1.TypeUrl,
		Value:   rawStatus,
	}
	resource := &v1.ClusterIngress{
		ClusterIngressSpec:   spec,
		ClusterIngressStatus: status,
	}
	resource.Metadata = kubeutils.FromKubeMeta(ingress.ObjectMeta)
	return resource, nil
}

func watchPod(c *KnativeController) {
	for {
		pods, err := c.KubeClientSet.CoreV1().Pods("ledboot").List(metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		c.Logger.Infof("There are %d pods in the cluster\n", len(pods.Items))
		time.Sleep(1 * time.Second)
	}
}
