package clusteringress

import (
	"github.com/hbagdi/go-kong/kong"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	informers "github.com/knative/serving/pkg/client/informers/externalversions/networking/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/networking/v1alpha1"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	kongCtl "github.com/ledboot/knative-cluster-ingress/pkg/reconiler/kong"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"strings"
	"time"
)

const (
	ServerTAG string = "Serverless.Service"
	RouteTAG  string = "Serverless.Route"
)

type KnativeController struct {
	*reconiler.Base
	clusterIngressLister listers.ClusterIngressLister
	kongServiceCtl       kongCtl.ServiceController
	queue                workqueue.RateLimitingInterface
}

func NewController(opts reconiler.Options, clusteringressInformer informers.ClusterIngressInformer) *KnativeController {
	kc := kongCtl.NewServiceController(opts.KongClient, opts.Logger)
	c := &KnativeController{
		Base:                 reconiler.NewBase(opts),
		kongServiceCtl:       *kc,
		clusterIngressLister: clusteringressInformer.Lister(),
	}

	clusterEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ing := obj.(v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(ing, v1.EventTypeNormal, "CREATE", fmt.Sprintf("clusteringress %s/%s", ing.Namespace, ing.Name))
			c.Logger.Infof("clusteringress add %v", ing)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			curIng := newObj.(v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(&curIng, v1.EventTypeNormal, "DELETE", fmt.Sprintf("clusteringress %s/%s", curIng.Namespace, curIng.Name))
			c.Logger.Infof("clusteringress update %v", curIng)
		},
		DeleteFunc: func(obj interface{}) {
			ing := obj.(v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(&ing, v1.EventTypeNormal, "DELETE", fmt.Sprintf("clusteringress %s/%s", ing.Namespace, ing.Name))
			c.Logger.Infof("clusteringress del %v", ing)
		},
	}

	clusteringressInformer.Informer().AddEventHandler(clusterEventHandler)
	return c
}

func (c *KnativeController) Start() {
	go c.watchClusterIngress()
	//go watchPod(c)
}

func (c *KnativeController) watchClusterIngress() {
	w, err := c.ServingClientSet.NetworkingV1alpha1().ClusterIngresses().Watch(metav1.ListOptions{})
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
		if service, err := c.kongServiceCtl.Get(resource.Name); err == nil {
			resource.Service = *service
			c.kongServiceCtl.Delete(*resource)
		}
	case watch.Modified:
		if service, err := c.kongServiceCtl.Get(resource.Name); err == nil {
			resource.Service = *service
			c.kongServiceCtl.Update(*resource)
		}
	case watch.Added:
		c.kongServiceCtl.Create(*resource)
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

func fromKube(ingress *v1alpha1.ClusterIngress) (*kongCtl.Service, error) {

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
