package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers/apps/v1beta2"
	"k8s.io/client-go/kubernetes"
	v1beta22 "k8s.io/client-go/listers/apps/v1beta2"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientset        kubernetes.Clientset
	deploymentLister v1beta22.DeploymentLister
	informerSynced   cache.InformerSynced
	queue            workqueue.RateLimitingInterface
}

func newController(clientset kubernetes.Clientset, depInformer v1beta2.DeploymentInformer) *controller {
	c := &controller{
		clientset:        clientset,
		deploymentLister: depInformer.Lister(),
		informerSynced:   depInformer.Informer().HasSynced,
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDel,
		})

	return c
}

func (c *controller) run(ch <-chan struct{}) {
	fmt.Println("starting controller")
	if !cache.WaitForCacheSync(ch, c.informerSynced) {
		// wait for the informer to fill cache before starting
		fmt.Println("waiting for cache to sync")
	}
}

func (c *controller) worker() {
	for c.processItem() {

	}
}

func (c *controller) processItem() bool {
	// get item by key from queue
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	defer c.queue.Forget(item)
	// return item from cache by key
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil {
		fmt.Printf("failed to get item from cache%s\n", err)
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Printf("sputting key into namespace and name failed %s\n", err)
		return false
	}

	ctx := context.Background()
	_, err = c.clientset.AppsV1().Deployments(ns).Get(ctx, name, v1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Println("not implemented!")
	}

	return false
}

func (c *controller) handleAdd(obj any) {
	fmt.Println("add was called")
	c.queue.Add(obj)
}

func (c *controller) handleDel(obj any) {
	fmt.Println("del was called")
	c.queue.Add(obj)
}
