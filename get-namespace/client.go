package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"path/filepath"
)

func getNamespaces() []list.Item {

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("error getting user home dir: %v\n", err)
		os.Exit(1)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)

	if err != nil {
		fmt.Printf("Error getting kubernetes config: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)

	if err != nil {
		fmt.Printf("error getting kubernetes config: %v\n", err)
		os.Exit(1)
	}

	ns, _ := clientset.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	var items []list.Item
	for _, nn := range ns.Items {
		items = append(items, item(nn.Name))
	}
	return items
}

func switchContext(ctx string) {
	cmd := exec.Command("kubectl", "config", "set-context", "--current", fmt.Sprintf("--namespace=%s", ctx))
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
