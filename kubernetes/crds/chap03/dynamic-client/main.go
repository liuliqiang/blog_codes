package main

import (
	"os"

	"github.com/liuliqiang/log4go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// created by: https://liqiang.io
	kubeconfig := "/etc/rancher/k3s/k3s.yaml"
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	client, err := dynamic.NewForConfig(config)

	// created by: https://liqiang.io
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	namespace := "default"
	resp, err := client.Resource(gvr).
		Namespace(namespace).Get("node-hello", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	// created by: https://liqiang.io
	name, found, err := unstructured.NestedString(resp.Object, "metadata", "name")
	if err != nil {
		panic(err.Error())
	}
	if !found {
		log4go.Info("Not found")
		os.Exit(-1)
	}
	log4go.Info("get a deployment name: %s", name)
}
