package main

import (
	"flag"
	"fmt"
	informers2 "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

func main() {
	kubeCfg := flag.String("kubeconfig", "~/.kube/config", "Kubeconfig location.")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeCfg)
	if err != nil {
		fmt.Printf("error building config from files %s\n", err)
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("failed to get incluster config %s\n", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("failed when attempting to build clientset %s ", err)
		os.Exit(1)
	}

	ch := make(chan struct{})
	informers := informers2.NewSharedInformerFactory(clientset, 10*time.Minute)

}
