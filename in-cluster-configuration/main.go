package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

const (
	fakePod = "i-dont-exist"
)

func main() {
	// create a config object which uses the service account Kubernetes gives to pods.
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("failed to create incluster config %v", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("failed to build clientset from config %v", err)
	}

	for {
		pods, err := clientset.CoreV1().Pods(v1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("failed to list pods %v", err)
			os.Exit(1)
		}

		fmt.Printf("There are %d pods in the cluster", len(pods.Items))

		// introduce an error
		_, err = clientset.CoreV1().Pods("default").Get(context.TODO(), fakePod, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod %s not found in default namespace\n", fakePod)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("Found %s pod in default namespace\n", fakePod)
		}
		time.Sleep(10 * time.Second)
	}
}
