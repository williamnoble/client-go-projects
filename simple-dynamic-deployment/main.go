package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	"os"
	"path/filepath"
)

func int32Ptr(i int32) *int32 { return &i }

const (
	deploymentName string = "wills-demo-deployment"
)

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

	// create client from rest config
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to build dynamic client from configuration", err)
		os.Exit(1)
	}

	// We don't have a structure type thus we need some way to identify our resource in the cluster, hence we
	// identify via GVR. We use this object identifying the resource to perform CRUD actions.
	deploymentResource := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	fmt.Println("Creating deployment")
	// TODO: Fix
	result, err := client.Resource(deploymentResource).Namespace(metav1.NamespaceDefault).Create(context.TODO(), unstructuredNginxDeployment(), metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("failed to create %s %w", deploymentName, err)
		os.Exit(1)
	}
	fmt.Printf("created deployment %q\n", result.GetName())

	// prompt to continue
	promptReturnToContinue()

	// lets try to update our deployment
	fmt.Println("Updating deployment")

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get latest version of deployment before attempting to update
		result, getErr := client.Resource(deploymentResource).Namespace(metav1.NamespaceDefault).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if getErr != nil {
			fmt.Println("Failed to get latest version of our deployment", err)
			os.Exit(1)
		}

		// update replicas - not as straight forward as we can't reference field directly
		if err := unstructured.SetNestedField(result.Object, int64(2), "spec", "replicas"); err != nil {
			fmt.Printf("Failed to update deployment via unstructure.SetNestedField %w", err)
			os.Exit(1)
		}

		_, updateErr := client.Resource(deploymentResource).Namespace(apiv1.NamespaceDefault).Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		fmt.Printf("Encounted an error when attempting to update deployment %v", err)
		os.Exit(1)
	}

	promptReturnToContinue()

	// list Deployments
	fmt.Printf("Listing deployments in namespace %s\n", metav1.NamespaceDefault)
	list, err := client.Resource(deploymentResource).Namespace(apiv1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list deployment in %s namespace.\n", metav1.NamespaceDefault)
		os.Exit(1)
	}
	for _, l := range list.Items {
		replicas, found, err := unstructured.NestedInt64(l.Object, "spec", "replicas")
		if err != nil || !found {
			fmt.Printf("Replicas not found for %s\n", deploymentName)
			continue
		}
		fmt.Printf("* %s (%d replicas)\n", l.GetName(), replicas)
	}
	promptReturnToContinue()

	// Delete Deployment
	fmt.Println("Deleting deployment")
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	if err := client.Resource(deploymentResource).Namespace(apiv1.NamespaceDefault).Delete(context.TODO(), deploymentName, deleteOptions); err != nil {
		fmt.Printf("Failed to delete deployment %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Deployment deleted :)")

}

func unstructuredNginxDeployment() *unstructured.Unstructured {
	d := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": deploymentName,
			},
			"spec": map[string]interface{}{
				"replicas": 1,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "demo",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "demo",
						},
					},

					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "web",
								"image": "nginx:1.12",
								"ports": []map[string]interface{}{
									{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": 80,
									},
								},
							},
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
