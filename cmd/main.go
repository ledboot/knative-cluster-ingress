package main

import (
	"flag"
	"github.com/hbagdi/go-kong/kong"
	informers "github.com/knative/serving/pkg/client/informers/externalversions"
	"github.com/ledboot/knative-cluster-ingress/pkg/configmap"
	"github.com/ledboot/knative-cluster-ingress/pkg/logging"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler/clusteringress"
	"github.com/ledboot/knative-cluster-ingress/pkg/signals"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

const (
	component = "ClusterIngress"
)

var (
	masterURL    = flag.String("master", "https://192.168.64.5:8443", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	kubeconfig   = flag.String("kubeconfig", "/Users/gwynn/.kube/config", "Path to a kubeconfig. Only required if out-of-cluster.")
	kongAdminURL = flag.String("kong_admin_url", "http://192.168.64.5:30936", "Kong Admin URL")
)

func main() {
	flag.Parse()
	stopCh := signals.SetupSignalHandler()
	//loggingConfigMap, err := configmap.Load("/etc/config-logging")
	loggingConfigMap, err := configmap.Load("/Users/gwynn/go/src/github.com/ledboot/knative-cluster-ingress/config-logging")
	if err != nil {
		log.Fatalf("Error loading logging configuration: %v", err)
	}
	loggingConfig, err := logging.NewConfigFromMap(loggingConfigMap)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, _ := logging.NewLogger(loggingConfig, component)
	defer logger.Sync()

	//创建k8sclient、knative client

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		logger.Fatalw("Error building kubeconfig", zap.Error(err))
	}
	opt := reconiler.NewOptionOrDie(cfg, logger, stopCh)

	//make kong client
	kongClient, err := kong.NewClient(kongAdminURL, nil)
	if err != nil {
		logger.Fatalf("make kong client error :", err)
	}
	root, err := kongClient.Root(nil)
	if err != nil {
		logger.Fatalf("can not connect kong admin :", err)
	}
	opt.KongClient = kongClient
	logger.Infof("kong version : %s", root["version"].(string))

	servingInformerFactory := informers.NewSharedInformerFactory(opt.ServingClientSet, opt.ResyncPeriod)
	clusterIngressInformer := servingInformerFactory.Networking().V1alpha1().ClusterIngresses()

	controller := clusteringress.NewController(opt, clusterIngressInformer)

	clusterIngressInformer.Informer().Run(stopCh)

	controller.Start()
	<-stopCh
}
