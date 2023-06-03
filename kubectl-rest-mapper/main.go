package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"os"
	"text/template"
)

func main() {
	// the resource to lookup via rest-mapper
	resource := "deployment"

	// configFlags required for generating a new Rest Config
	configFlag := genericclioptions.NewConfigFlags(true)
	matchVersionFlags := util.NewMatchVersionFlags(configFlag)

	// NewFactory has a method ToRESTMapper which allows clients to map resources to kind,
	// and map kind and version to interfaces for manipulating those objects
	m, err := util.NewFactory(matchVersionFlags).ToRESTMapper()
	if err != nil {
		fmt.Printf("failed to build new factory %v", err)
		os.Exit(1)
	}

	groupVersionResource, err := m.ResourceFor(schema.GroupVersionResource{
		Resource: resource,
	})

	if err != nil {
		fmt.Printf("failed to get GVR for %s resource %v \n", resource, err)
		return
	}

	data := struct {
		Group    string
		Version  string
		Resource string
	}{
		Group:    groupVersionResource.Group,
		Version:  groupVersionResource.Version,
		Resource: groupVersionResource.Resource,
	}

	t := template.Must(template.ParseFiles("./template.tpl"))
	if err != nil {
		fmt.Printf("failed to parse template file: %v\n", err)
		os.Exit(1)
	}
	err = t.Execute(os.Stdout, data)
	if err != nil {
		fmt.Printf("failed to render template to stdout %v\n", err)
		os.Exit(1)
	}

}
