package main

import (
	"flag"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giolekva/pcloud/events"

	"github.com/golang/glog"
)

var kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file.")
var apiAddr = flag.String("api_addr", "", "PCloud API server address.")
var objectStoreAddr = flag.String("object_store_addr", "", "S3 compatible object store address.")

func getKubeConfig() (*rest.Config, error) {
	if *kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		return rest.InClusterConfig()
	}
}

func main() {
	flag.Parse()
	config, err := getKubeConfig()
	if err != nil {
		glog.Fatalf("Could not initialize Kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Could not create Kubernetes API client: %v", err)
	}
	pods := clientset.CoreV1().Pods("default")
	eventStore := events.NewGraphQLClient(*apiAddr)
	events.NewSingleEventAtATimeProcessor(
		eventStore, pods, *apiAddr, *objectStoreAddr).Start()
}
