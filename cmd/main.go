package main

import (
	"flag"
	"github.com/ledboot/knative-cluster-ingress/pkg/configmap"
	"github.com/ledboot/knative-cluster-ingress/pkg/logging"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler"
	"github.com/ledboot/knative-cluster-ingress/pkg/reconiler/api/clusteringress"
	"github.com/ledboot/knative-cluster-ingress/pkg/signals"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

const (
	component = "controller"
)

var (
	masterURL  = flag.String("master", "https://192.168.64.5:8443", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	kubeconfig = flag.String("kubeconfig", "/Users/gwynn/.kube/config", "Path to a kubeconfig. Only required if out-of-cluster.")
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
	controller := clusteringress.NewController(opt)
	logger.Info(controller.KnativeClientSet)
	controller.Start()
	<-stopCh
}
