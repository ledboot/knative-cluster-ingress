package reconiler

import (
	knativeclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"time"
)

var resetPeriod = 30 * time.Second

type Options struct {
	KubeClientSet    kubernetes.Interface
	KnativeClientSet knativeclientset.Interface
	Logger           *zap.SugaredLogger
	StopChannel      <-chan struct{}
	ResyncPeriod     time.Duration
	Recorder         record.EventRecorder
}

type Base struct {
	KubeClientSet    kubernetes.Interface
	KnativeClientSet knativeclientset.Interface
	Logger           *zap.SugaredLogger
	Recorder         record.EventRecorder
}

func NewOptionOrDie(cfg *rest.Config, logger *zap.SugaredLogger, stopCh <-chan struct{}) Options {
	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	knativeClient := knativeclientset.NewForConfigOrDie(cfg)

	return Options{
		KubeClientSet:    kubeClient,
		KnativeClientSet: knativeClient,
		Logger:           logger,
		StopChannel:      stopCh,
		ResyncPeriod:     resetPeriod,
	}
}

func NewBase(opt Options) *Base {
	recorder := opt.Recorder
	logger := opt.Logger
	if recorder == nil {
		logger.Debug("Creating event broadcaster")
		eventBroadcaster := record.NewBroadcaster()
		watchs := []watch.Interface{
			eventBroadcaster.StartLogging(logger.Named("event-broadcaster").Infof),
			eventBroadcaster.StartRecordingToSink(
				&typedcorev1.EventSinkImpl{Interface: opt.KubeClientSet.CoreV1().Events("")}),
		}
		recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "knative-cluster-ingress"})
		go func() {
			<-opt.StopChannel
			for _, w := range watchs {
				w.Stop()
			}
		}()
	}
	base := &Base{
		KubeClientSet:    opt.KubeClientSet,
		KnativeClientSet: opt.KnativeClientSet,
		Logger:           opt.Logger,
	}
	return base
}
