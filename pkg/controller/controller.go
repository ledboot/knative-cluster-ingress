package controller

import (
	"context"
	"fmt"
	"github.com/ledboot/knative-cluster-ingress/pkg/logging"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sync"
)

var (
	DefaultThreadPerController = 2
)

type EventType string

type Event struct {
	Type   EventType
	Object interface{}
	Key    types.UID
}

type Informer interface {
	Run(<-chan struct{})
	HasSynced() bool
}

type Reconciler interface {
	Reconcile(ctx context.Context, event Event) error
}

type Impl struct {
	Reconciler Reconciler
	WorkQueue  workqueue.RateLimitingInterface
	Logger     *zap.SugaredLogger
}

func StartInformers(stopCh <-chan struct{}, informers ...Informer) error {
	for _, informer := range informers {
		informer := informer
		go informer.Run(stopCh)
	}

	for i, informer := range informers {
		if ok := cache.WaitForCacheSync(stopCh, informer.HasSynced); !ok {
			return fmt.Errorf("Failed to wait for cache at index %d to sync", i)
		}
	}
	return nil
}

func StartAll(stopCh <-chan struct{}, controllers ...*Impl) {
	wg := sync.WaitGroup{}
	for _, c := range controllers {
		wg.Add(1)
		go func(c *Impl) {
			defer wg.Done()
			c.Run(DefaultThreadPerController, stopCh)
		}(c)
	}
	wg.Wait()

}

func NewImpl(r Reconciler, logger *zap.SugaredLogger, workQueueName string) *Impl {
	return &Impl{
		Reconciler: r,
		Logger:     logger,
		WorkQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			workQueueName,
		),
	}
}

func (c *Impl) Enqueue(event *Event) {
	c.WorkQueue.Add(event)
}

func (c *Impl) processNextWorkItem() bool {
	obj, shutdown := c.WorkQueue.Get()
	if shutdown {
		c.Logger.Info("workqueue shutdown")
		return false
	}
	event := obj.(*Event)
	defer c.WorkQueue.Done(event)

	logger := c.Logger.With(zap.String(logging.Key, ""))
	ctx := logging.WithLogger(context.TODO(), logger)

	if err := c.Reconciler.Reconcile(ctx, *event); err != nil {
		logger.Infof("Reconcile failed.")
		return true
	}

	c.WorkQueue.Forget(event)
	logger.Infof("Reconcile succeeded.")
	return true
}

func (c *Impl) Run(threadiness int, stopCh <-chan struct{}) error {

	wg := sync.WaitGroup{}
	defer wg.Wait()
	defer c.WorkQueue.ShutDown()

	logger := c.Logger

	for i := 0; i < threadiness; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c.processNextWorkItem() {
			}
		}()
	}
	logger.Info("Started workers")
	<-stopCh
	logger.Info("Shutting down workers")

	return nil
}
