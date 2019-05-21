package clusteringress

import (
	"github.com/hbagdi/go-kong/kong"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	kongCtl "github.com/ledboot/knative-cluster-ingress/pkg/reconiler/kong"
	"strings"
)

func Filter() func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if object, ok := obj.(*v1alpha1.ClusterIngress); ok {
			if _, ok := object.ObjectMeta.Labels[RouteLabelKey]; ok {
				return true
			}
		}
		return false
	}
}

//func (c *KnativeController) watchClusterIngress() {
//	w, err := c.ServingClientSet.NetworkingV1alpha1().ClusterIngresses().Watch(metav1.ListOptions{})
//	if err != nil {
//		c.Logger.Fatal(err)
//	}
//	for {
//		select {
//		case event := <-w.ResultChan():
//			switch event.Type {
//			case watch.Error:
//				c.Logger.Errorf("error during watch: %v", event)
//			default:
//				c.listClusterIngress(event)
//			}
//
//		}
//	}
//}

//func (c *KnativeController) listClusterIngress(event watch.Event) {
//	ingressObjList, err := c.KnativeClientSet.NetworkingV1alpha1().ClusterIngresses().List(metav1.ListOptions{})
//	if err != nil {
//		c.Logger.Errorf("listing ingressObjs at cluster level get error : %s", err)
//	}
//	if event.Object == nil {
//		return
//	}
//	cl := event.Object.(*v1alpha1.ClusterIngress)
//	resource, err := fromKube(cl)
//	c.Logger.Info("ingress %v:", resource)
//	if err != nil {
//		c.Logger.Errorf("format ingerss error : %+v", err)
//		return
//	}
//	c.Logger.Infof("resource name : %s", cl.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend.ServiceName)
//	switch event.Type {
//	case watch.Deleted:
//		if service, err := c.KongServiceCtl.Get(resource.Name); err == nil {
//			resource.Service = *service
//			c.KongServiceCtl.Delete(*resource)
//		}
//	case watch.Modified:
//		if service, err := c.KongServiceCtl.Get(resource.Name); err == nil {
//			resource.Service = *service
//			c.KongServiceCtl.Update(*resource)
//		}
//	case watch.Added:
//		c.KongServiceCtl.Create(*resource)
//	}
//}

func diffClusterIngress(oldObj, newOjb *v1alpha1.ClusterIngress) bool {
	rawOldSpec := oldObj.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend
	rawNewSpec := newOjb.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend
	if rawOldSpec.ServiceName == rawNewSpec.ServiceName {
		return false
	}
	return true

}

func fromKube(ingress *v1alpha1.ClusterIngress) (*kongCtl.Service, error) {

	routeLabel := ingress.ObjectMeta.Labels[RouteLabelKey]
	namespaceLabel := ingress.ObjectMeta.Labels[RouseNamespaceLabelKes]

	serviceBackend := ingress.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend
	//serviceName := kong.String(serviceBackend.ServiceName)
	serviceName := kong.String(strings.Join([]string{routeLabel, namespaceLabel}, "-"))
	serviceHost := kong.String(strings.Join([]string{routeLabel, namespaceLabel, "svc.cluster.local"}, "."))
	servicePort := kong.Int(serviceBackend.ServicePort.IntValue())
	routeHosts := kong.StringSlice(ingress.Spec.Rules[0].Hosts...)
	routeProtocols := kong.StringSlice("http", "https")
	routeName := kong.String(strings.Join([]string{routeLabel, "-route"}, ""))
	appServiceTag := ingress.Spec.Rules[0].Hosts[0]

	kongService := &kong.Service{
		Host:           serviceHost,
		Name:           serviceName,
		Port:           servicePort,
		ConnectTimeout: kong.Int(60000),
		ReadTimeout:    kong.Int(60000),
		WriteTimeout:   kong.Int(60000),
		Retries:        kong.Int(5),
		Tags:           kong.StringSlice(ServerTAG, appServiceTag),
	}
	kongRoute := &kong.Route{
		Name:      routeName,
		Hosts:     routeHosts,
		Protocols: routeProtocols,
		Tags:      kong.StringSlice(RouteTAG),
	}
	service := &kongCtl.Service{
		Service: *kongService,
	}
	route := &kongCtl.Route{
		Route: *kongRoute,
	}
	service.Routes = append(service.Routes, *route)
	//if err != nil {
	//	return nil, err
	//}
	//spec := &v1.Any{
	//	TypeUrl: v1.TypeUrl,
	//	Value:   rawSpec,
	//}
	//
	//rawStatus, err := json.Marshal(ingress.Status)
	//if err != nil {
	//	return nil, err
	//}
	//
	//status := &v1.Any{
	//	TypeUrl: v1.TypeUrl,
	//	Value:   rawStatus,
	//}
	//resource := &v1.ClusterIngress{
	//	ClusterIngressSpec:   spec,
	//	ClusterIngressStatus: status,
	//}
	//resource.Metadata = kubeutils.FromKubeMeta(ingress.ObjectMeta)
	return service, nil
}
