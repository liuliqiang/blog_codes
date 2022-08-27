package main

import (
	"github.com/liuliqiang/log4go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// created by: https://liqiang.io
	kubeconfig := "/etc/rancher/k3s/k3s.yaml"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// created by: https://liqiang.io
	namespace := "default"
	deploy, err := client.AppsV1().Deployments(namespace).Get("node-hello", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	log4go.Info("get a deployment name: %s", deploy.Name)
}
