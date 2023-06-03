package main

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

// Controller holds the state for our controller. cache.Controller is a low-level controller which a) pumps objects from ListerWatcher
// to our work queue, b) pop items from the work queue for processing.
type Controller struct {
	// extend store via indices
	indexer cache.Indexer
	// a rate limited queue
	queue workqueue.RateLimitingInterface
	// create a low-level controller
	informer cache.Controller
}

func NewController(indexer cache.Indexer, rateLimitingInterface workqueue.RateLimitingInterface, informer cache.Controller) *Controller {
	return &Controller{
		indexer:  indexer,
		queue:    rateLimitingInterface,
		informer: informer,
	}
}

func (c *Controller) processNextItem() bool {
	queueKey, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	// finished processing current key
	defer c.queue.Done(queueKey)
	// Get returns item interface{} but we want a string
	err := c.syncToStdout(queueKey.(string))
	c.handleError(err, queueKey)
	return true
}

func (c *Controller) handleError(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v:%v", key, err)
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	runtime.HandleError(err)
	klog.Info("Dropping pod %q out of the queue: %v", key, err)
}

// syncToStdout is where our business logic lives.
// TODO: Remember, don't include retry logic as part of business logic!!
func (c *Controller) syncToStdout(key string) error {
	// returns an accumulator
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Failed to fetch key %s from store %w ", key, err)
		return err
	}
	if !exists {
		fmt.Printf("Pod %s does not exist anymore\n", key)
	} else {
		// GetByKey returns item interface{} hence type assert as Pod
		fmt.Printf("Sync/Add/Update for pod %s\n", obj.(*v1.Pod).GetName())
	}
	return nil
}

func (c *Controller) RunWorker() {
	for c.processNextItem() {
	}
}

// Run beings watching and syncing
func (c *Controller) Run(numOfWorkers int, stopChan chan struct{}) {
	defer runtime.HandleCrash()

	// the thing which writes to the queue do the shutdown
	defer c.queue.ShuttingDown()

	klog.Info("Starting pod controller")
	go c.informer.Run(stopChan)

	if !cache.WaitForCacheSync(stopChan, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < numOfWorkers; i++ {
		go wait.Until(c.RunWorker, time.Second, stopChan)
	}

	<-stopChan

	klog.Info("Stopping Pod Controller")
}
