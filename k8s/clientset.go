package k8s

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

var (
	clientset *kubernetes.Clientset
    config *restclient.Config
)

func InitClientset() {
	var kubeconfigPath *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfigPath = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	if err != nil {
		panic(err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
}

func GetConfig() *restclient.Config {
	return config
}

func GetClientset() *kubernetes.Clientset {
	return clientset
}


