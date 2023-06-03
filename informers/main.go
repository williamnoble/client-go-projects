package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func main() {

	home, err := os.UserHomeDir()
	kubeConfigFile := path.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		fmt.Printf("failed to build rest config %s\n", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(config)

	if err != nil {
		fmt.Printf("failed when attempting to create new clientset %s\n", err)
	}

	namespaces, err := client.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
	if errors.IsNotFound(err) {
		log.Fatal("No namespace in the cluster", err)
	} else if err != nil {
		log.Fatal("Failed to fetch namespaces in the cluster", err)
	}

	// clientset sanity check
	var nsList []string
	for _, namespace := range namespaces.Items {
		nsList = append(nsList, namespace.Name)

	}
	fmt.Printf("client sanity check, listing namespaces: %s", strings.Join(nsList, ", "))

	// create a configMap
	configMapOne := createConfigMap(client)
	fmt.Printf("executed create config map for %s\n", configMapOne.Name)
	// create shared informer factory, watch default namespace
	factory := informers.NewSharedInformerFactoryWithOptions(client, 10*time.Minute, informers.WithNamespace("default"))
	// create a configmap informer
	configMapInformer := factory.Core().V1().ConfigMaps()
	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj interface{}, isInInitialList bool) {
			configMapFromEvent := obj.(*v1.ConfigMap)
			if strings.Contains(configMapFromEvent.Name, "will") {
				fmt.Printf("informer event, config map added: %s\n", configMapFromEvent.Name)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			configMapFromEvent := oldObj.(*v1.ConfigMap)
			fmt.Printf("informer event, updated config map: %s\n", configMapFromEvent.Name)
		},
		DeleteFunc: func(obj interface{}) {
			configMapFromEvent := obj.(*v1.ConfigMap)
			fmt.Printf("informer event, deleted config map %s\n", configMapFromEvent.Name)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	factory.Start(ctx.Done())

	for informerType, ok := range factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			panic(fmt.Sprintf("Failed to sync cache for %v", informerType))
		}
	}

	selector, _ := labels.Parse("name==william")

	list, err := configMapInformer.Lister().List(selector)
	if err != nil {
		fmt.Printf("failed to get selected configmap by label: %s\n", list)
	}

	for _, l := range list {
		fmt.Println("found ", l.Name)
	}

	fmt.Println("Cleaning up...")
	deleteConfigMap(client, configMapOne)

}

func createConfigMap(client kubernetes.Interface) *v1.ConfigMap {
	c := &v1.ConfigMap{Data: map[string]string{"name": "william"}}
	c.Namespace = v1.NamespaceDefault
	c.GenerateName = "will-informer-example-"
	c.SetLabels(setLabels())

	res, err := client.CoreV1().ConfigMaps(v1.NamespaceDefault).Create(context.TODO(), c, v12.CreateOptions{})

	if err != nil {
		fmt.Printf("failed when attempting to create configmap %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created config-map %q in %q namespace\n", res.Name, res.Namespace)
	return res
}

func deleteConfigMap(client kubernetes.Interface, c *v1.ConfigMap) {
	err := client.CoreV1().ConfigMaps(c.GetNamespace()).Delete(context.TODO(), c.GetName(), v12.DeleteOptions{})
	if err != nil {
		fmt.Printf("failed to delete config map %s", c.Name)
	}

	fmt.Printf("Deleted configmap %s", c.GetName())
}

func setLabels() map[string]string {
	m := map[string]string{
		"name": "william",
		"bar":  "baz",
	}
	return m
}
