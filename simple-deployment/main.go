package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	"os"
	"path/filepath"
)

func int32Ptr(i int32) *int32 { return &i }

func main() {

	// fetch kubeconfig
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolue path to kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolue path to kubeconfig")
	}
	flag.Parse()

	// build rest config
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println("failed to build a rest configuration from kubeconfig", err)
		os.Exit(1)
	}

	// create clientset from rest config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to bulid clientset from configuration", err)
		os.Exit(1)
	}

	// from clientset create a deployments client, create a deployment object
	deploymentClient := clientset.AppsV1().Deployments(metav1.NamespaceDefault)
	fmt.Println("Creating deployment")
	result, err := deploymentClient.Create(context.TODO(), structuredNginxDeployment(), metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("failed to create %s deployment %v", structuredNginxDeployment().Name, err)
		os.Exit(1)
	}
	fmt.Printf("created deployment %q\n", result.GetObjectMeta().GetName())

	fmt.Println("Object version", result.GetObjectMeta().GetGeneration())
	// prompt to continue
	promptReturnToContinue()

	// lets try to update our deployment
	fmt.Println("Updating deployment")

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get latest verion of deployment before attempting to update
		result, getErr := deploymentClient.Get(context.TODO(), structuredNginxDeployment().GetName(), metav1.GetOptions{})
		if getErr != nil {
			fmt.Println("Failed to get latest version of our deployment", err)
			os.Exit(1)
		}

		result.Spec.Replicas = int32Ptr(2)
		_, updateErr := deploymentClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		fmt.Printf("Encounted an error when attempting to update deployment %v", err)
		os.Exit(1)
	}

	promptReturnToContinue()

	// list Deployments
	fmt.Printf("Listing deployments in namespace %s\n", metav1.NamespaceDefault)
	list, err := deploymentClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list deployment in %s namespace.\n", metav1.NamespaceDefault)
		os.Exit(1)
	}
	for _, l := range list.Items {
		fmt.Printf("* %s (%d replicas)\n", l.Name, *l.Spec.Replicas)
	}
	promptReturnToContinue()

	// Delete Deployment
	fmt.Println("Deleting deployment")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentClient.Delete(context.TODO(), "wills-demo-deployment", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		fmt.Printf("Failed to delete deployment %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Deployment deleted :)")

}

func structuredNginxDeployment() *appsv1.Deployment {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "wills-demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: getLabels(),
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getLabels(),
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:latest",
							Ports: getPorts(),
						},
					},
				},
			},
		},
	}
	return d
}

func getLabels() map[string]string {
	labels := map[string]string{
		"app": "demo",
	}
	return labels
}

func getPorts() []apiv1.ContainerPort {
	p := []apiv1.ContainerPort{
		{
			Name:          "http",
			Protocol:      apiv1.ProtocolTCP,
			ContainerPort: 80,
		},
	}
	return p
}

func promptReturnToContinue() {
	fmt.Printf("Press the return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Encountered an error when scanning")
		os.Exit(1)
	}

	fmt.Println()
}
