package main

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"os"
	"path"
)

func main() {
	home, err := os.UserHomeDir()
	kubeConfigFile := path.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		klog.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	// let's create a ListWatch for the specified client. ListWatch knows how to list and watch a set of apiserver resources
	// we're using a `cache` to reduce server calls.A Reflector watches the server and updates the Store.
	// The Store provides both a cache and a FIFO queue.
	// remember: fields.Everything() as a field selector
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())

	// create a work queue, we want to bind this queue to a cache via an informer.
	// when the cache is updated, a pod key is added to the work queue.
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// NewIndexerInformer returns an Indexer and a Controller, it populates the index and provides event notifications.
	// podListWatcher: what resource do we want to be informed about? (in this case pods in the default namespace).
	// objType: The object we expect to receive (in our case, it's a v1::Pod).
	// eventHandlers: callback functions to receive notifications.
	// indexers: indexer for the received object type.
	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewController(indexer, queue, informer)

	indexer.Add(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mypod",
			Namespace: v1.NamespaceDefault,
		},
	})

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)
	select {}
}
