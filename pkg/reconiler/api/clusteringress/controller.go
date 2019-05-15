package clusteringress

import (
	"github.com/hbagdi/go-kong/kong"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	kongCtl "github.com/ledboot/knative-cluster-ingress/pkg/reconiler/api/kong"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"strings"
	"time"
)

const (
	ServerTAG string = "Serverless.Service"
	RouteTAG  string = "Serverless.Route"
)

type KnativeController struct {
	*reconiler.Base
	kongCtl.ServiceController
}

func NewController(opts reconiler.Options) *KnativeController {
	kc := kongCtl.NewServiceController(opts.KongClient, opts.Logger)
	c := &KnativeController{
		Base:              reconiler.NewBase(opts),
		ServiceController: *kc,
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
				c.listClusterIngress(event)
			}

		}
	}
}

func (c *KnativeController) listClusterIngress(event watch.Event) {
	//ingressObjList, err := c.KnativeClientSet.NetworkingV1alpha1().ClusterIngresses().List(metav1.ListOptions{})
	//if err != nil {
	//	c.Logger.Errorf("listing ingressObjs at cluster level get error : %s", err)
	//}
	if event.Object == nil {
		return
	}
	cl := event.Object.(*v1alpha1.ClusterIngress)
	resource, err := fromKube(cl)
	c.Logger.Info("ingress %v:", resource)
	if err != nil {
		c.Logger.Errorf("format ingerss error : %+v", err)
		return
	}
	c.Logger.Infof("resource name : %s", resource.Service.Name)
	switch event.Type {
	case watch.Deleted:
		if service, err := c.ServiceController.Get(resource.Name); err == nil {
			resource.Service = *service
			c.ServiceController.Delete(*resource)
		}
	case watch.Modified:
		if service, err := c.ServiceController.Get(resource.Name); err == nil {
			resource.Service = *service
			c.ServiceController.Update(*resource)
		}
	case watch.Added:
		c.ServiceController.Create(*resource)
	}

	//for _, ingressObj := range ingressObjList.Items {
	//
	//	//for i, r := range ingressObj.Spec.Rules {
	//	//	host := r.Hosts
	//	//	serviceName :=
	//	//}
	//	//c.KongClient.Services.Create()
	//	//c.KongClient.Routes.Create()
	//	resource, err := fromKube(&ingressObj)
	//	if err != nil {
	//		c.Logger.Errorf("format ingerss error : %s", err)
	//		continue
	//	}
	//	c.Logger.Infof("get ingress :", resource)
	//
	//	svc, err := c.KongClient.Services.Create(nil, &resource.Service)
	//	if err != nil {
	//		c.Logger.Errorf("create kong service fail v%", err)
	//		continue
	//	}
	//	c.Logger.Infof("create kong service successful v%", svc)
	//
	//	for _, r := range resource.Routes {
	//		r.Route.Service = svc
	//		route, err := c.KongClient.Routes.Create(nil, &r.Route)
	//		if err != nil {
	//			c.Logger.Errorf("create kong route fail v%", err)
	//			continue
	//		}
	//		c.Logger.Infof("create kong route successful v%", route)
	//	}
	//}

}

func fromKube(ingress *v1alpha1.ClusterIngress) (*v1.Service, error) {

	//rawSpec, err := json.Marshal(ingress.Spec)
	serviceBackend := &ingress.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend
	serviceName := kong.String(serviceBackend.ServiceName)
	serviceHost := kong.String(strings.Join([]string{serviceBackend.ServiceName, serviceBackend.ServiceNamespace, "svc.cluster.local"}, "."))
	servicePort := kong.Int(serviceBackend.ServicePort.IntValue())
	routeHosts := kong.StringSlice(ingress.Spec.Rules[0].Hosts...)
	routeProtocols := kong.StringSlice("http", "https")
	routeName := kong.String(strings.Join([]string{serviceBackend.ServiceName, "-route"}, ""))
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
	service := &v1.Service{
		Service: *kongService,
	}
	route := &v1.Route{
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
