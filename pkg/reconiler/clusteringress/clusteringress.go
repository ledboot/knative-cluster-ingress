package clusteringress

import (
	"context"
	"errors"
	"github.com/hbagdi/go-kong/kong"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	informers "github.com/knative/serving/pkg/client/informers/externalversions/networking/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/networking/v1alpha1"
	"github.com/ledboot/knative-cluster-ingress/pkg/controller"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	kongCtl "github.com/ledboot/knative-cluster-ingress/pkg/reconiler/kong"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerAgentName    = "clusteringress-controller"
	GroupName              = "serving.knative.dev"
	RouteLabelKey          = GroupName + "/route"
	RouseNamespaceLabelKes = GroupName + "/routeNamespace"

	ServerTAG string = "Serverless.Service"
	RouteTAG  string = "Serverless.Route"

	// CreateEvent event associated with new objects in an informer
	CreateEvent controller.EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent controller.EventType = "UPDATE"
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent controller.EventType = "DELETE"
)

type Reconciler struct {
	*reconiler.Base
	clusterIngressLister listers.ClusterIngressLister
	KongServiceCtl       *kongCtl.ServiceController
}

func NewController(opts reconiler.Options, clusteringressInformer informers.ClusterIngressInformer) *controller.Impl {
	kc := kongCtl.NewServiceController(opts.KongClient, opts.Logger)

	c := &Reconciler{
		Base:           reconiler.NewBase(opts, controllerAgentName),
		KongServiceCtl: kc,
	}

	impl := controller.NewImpl(c, c.Logger, "ClusterIngresses")

	clusterEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ing := obj.(*v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(obj.(runtime.Object), v1.EventTypeNormal, "CREATE", fmt.Sprintf("clusteringress %s/%s", ing.Namespace, ing.Name))
			//cache.MetaNamespaceIndexFunc(obj)
			event := &controller.Event{
				Type:   CreateEvent,
				Object: ing,
				Key:    ing.ObjectMeta.UID,
			}
			impl.Enqueue(event)
			c.Logger.Infof("clusteringress add %v", ing)

		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			curIng := newObj.(*v1alpha1.ClusterIngress)
			oldIng := oldObj.(*v1alpha1.ClusterIngress)
			//cache.MetaNamespaceIndexFunc(newObj)
			//c.Recorder.Eventf(&curIng, v1.EventTypeNormal, "DELETE", fmt.Sprintf("clusteringress %s/%s", curIng.Namespace, curIng.Name))
			if diffClusterIngress(oldIng, curIng) {
				c.Logger.Infof("clusteringress update %v", curIng)
				event := &controller.Event{
					Type:   UpdateEvent,
					Object: curIng,
					Key:    curIng.ObjectMeta.UID,
				}
				impl.Enqueue(event)
			}
		},
		DeleteFunc: func(obj interface{}) {
			ing := obj.(*v1alpha1.ClusterIngress)
			//c.Recorder.Eventf(&ing, v1.EventTypeNormal, "DELETE", fmt.Sprintf("clusteringress %s/%s", ing.Namespace, ing.Name))
			c.Logger.Infof("clusteringress del %v", ing)
			event := &controller.Event{
				Type:   DeleteEvent,
				Object: ing,
				Key:    ing.ObjectMeta.UID,
			}
			impl.Enqueue(event)
		},
	}
	clusteringressInformer.Informer().AddEventHandler(clusterEventHandler)

	return impl
}

func (c *Reconciler) Reconcile(ctx context.Context, event controller.Event) error {
	cl, ok := event.Object.(*v1alpha1.ClusterIngress)
	if ok {
		resource, err := fromKube(cl)
		if err != nil {
			c.Logger.Errorf("format ingerss error : %+v", err)
			return nil
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
		return nil
	}
	return errors.New("parse clusteringress fail")
}
