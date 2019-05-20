package clusteringress

import (
	"fmt"
	"github.com/hbagdi/go-kong/kong"
	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	informers "github.com/knative/serving/pkg/client/informers/externalversions/networking/v1alpha1"
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

	// CreateEvent event associated with new objects in an informer
	CreateEvent EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent EventType = "UPDATE"
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent EventType = "DELETE"
)

type KnativeController struct {
	*reconiler.Base
	clusteringressInformer informers.ClusterIngressInformer
	KongServiceCtl         kongCtl.ServiceController
	WorkQueue              workqueue.RateLimitingInterface
}

type EventType string

type Event struct {
	Type   EventType
	Object interface{}
}

func NewController(opts reconiler.Options, clusteringressInformer informers.ClusterIngressInformer) *KnativeController {
	kc := kongCtl.NewServiceController(opts.KongClient, opts.Logger)
	c := &KnativeController{
		Base:                   reconiler.NewBase(opts),
		KongServiceCtl:         *kc,
		clusteringressInformer: clusteringressInformer,
		WorkQueue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	clusterEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ing := obj.(*v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(obj.(runtime.Object), v1.EventTypeNormal, "CREATE", fmt.Sprintf("clusteringress %s/%s", ing.Namespace, ing.Name))
			//cache.MetaNamespaceIndexFunc(obj)
			event := &Event{
				Type:   CreateEvent,
				Object: ing,
			}
			c.WorkQueue.Add(event)
			c.Logger.Infof("clusteringress add %v", ing)

		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			curIng := newObj.(*v1alpha1.ClusterIngress)
			oldIng := oldObj.(*v1alpha1.ClusterIngress)
			//cache.MetaNamespaceIndexFunc(newObj)
			//c.Recorder.Eventf(&curIng, v1.EventTypeNormal, "DELETE", fmt.Sprintf("clusteringress %s/%s", curIng.Namespace, curIng.Name))
			if diffClusterIngress(oldIng, curIng) {
				c.Logger.Infof("clusteringress update %v", curIng)
				event := &Event{
					Type:   UpdateEvent,
					Object: curIng,
				}
				c.WorkQueue.Add(event)
			}
		},
		DeleteFunc: func(obj interface{}) {
			ing := obj.(*v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(&ing, v1.EventTypeNormal, "DELETE", fmt.Sprintf("clusteringress %s/%s", ing.Namespace, ing.Name))
			c.Logger.Infof("clusteringress del %v", ing)
			event := &Event{
				Type:   DeleteEvent,
				Object: ing,
			}
			c.WorkQueue.Add(event)
		},
	}

	//clusteringressInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
	//	Handler:    clusterEventHandler,
	//	FilterFunc: reconiler.LabelExistFilter(ClusterIngressKind),
	//})
	//clusteringressInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
	//	Handler:    clusterEventHandler,
	//	FilterFunc: Filter(),
	//})
	clusteringressInformer.Informer().AddEventHandler(clusterEventHandler)
	return c
}

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

func DeletionHandlingAccessor(obj interface{}) (*kmeta.Accessor, error) {
	accessor, ok := obj.(*kmeta.Accessor)
	if !ok {
		// To handle obj deletion, try to fetch info from DeletedFinalStateUnknown.
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return nil, fmt.Errorf("Couldn't get Accessor from tombstone %#v", obj)
		}
		accessor, ok = tombstone.Obj.(*kmeta.Accessor)
		if !ok {
			return nil, fmt.Errorf("The object that Tombstone contained is not of kmeta.Accessor %#v", obj)
		}
	}

	return accessor, nil
}

func (c *KnativeController) Start(stopCh <-chan struct{}) {
	go c.clusteringressInformer.Informer().Run(stopCh)
	//go c.watchClusterIngress()
	defer c.WorkQueue.ShutDown()

	for c.processNextWorkItem() {
	}
}

func (c *KnativeController) processNextWorkItem() bool {
	obj, shutdown := c.WorkQueue.Get()
	if shutdown {
		c.Logger.Info("shutdown")
		return false
	}
	event := obj.(*Event)
	cl, ok := event.Object.(*v1alpha1.ClusterIngress)
	if ok {
		resource, err := fromKube(cl)
		if err != nil {
			c.Logger.Errorf("format ingerss error : %+v", err)
			return true
		}
		c.Logger.Infof("process type: %s, event :%s", event.Type, cl.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend.ServiceName)
		switch event.Type {
		case CreateEvent:
			service, _ := c.KongServiceCtl.Get(resource.Name)
			if service == nil {
				c.KongServiceCtl.Create(*resource)
			}
		case UpdateEvent:
			if service, err := c.KongServiceCtl.Get(resource.Name); err == nil {
				resource.Service.ID = kong.String(*service.ID)
				c.KongServiceCtl.Update(*resource)
			}
		case DeleteEvent:
			if service, err := c.KongServiceCtl.Get(resource.Name); err == nil {
				resource.Service = *service
				c.KongServiceCtl.Delete(*resource)
			}
		}
	}
	return true
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
	c.Logger.Infof("resource name : %s", cl.Spec.Rules[0].HTTP.Paths[0].Splits[0].ClusterIngressBackend.ServiceName)
	switch event.Type {
	case watch.Deleted:
		if service, err := c.KongServiceCtl.Get(resource.Name); err == nil {
			resource.Service = *service
			c.KongServiceCtl.Delete(*resource)
		}
	case watch.Modified:
		if service, err := c.KongServiceCtl.Get(resource.Name); err == nil {
			resource.Service = *service
			c.KongServiceCtl.Update(*resource)
		}
	case watch.Added:
		c.KongServiceCtl.Create(*resource)
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
