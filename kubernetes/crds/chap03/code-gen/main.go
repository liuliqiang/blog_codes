package main

import (
	"github.com/liuliqiang/blog-demos/kubernetes/crds/chap03/code-gen/client/clientset/versioned"

	"github.com/liuliqiang/log4go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// created by: https://liqiang.io
	kubeconfig := "/etc/rancher/k3s/k3s.yaml"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	client, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// created by: https://liqiang.io
	namespace := "default"
	admin, err := client.AdminV1().Admins(namespace).Get("liuliqiang", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	log4go.Info("get a admin password is: %s", admin.Spec.Password)
}
