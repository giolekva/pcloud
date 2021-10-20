package main

import (
	"context"
	"flag"
	"io/ioutil"
	"path"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var configFile = flag.String("config", "", "Path to the homeserver.yaml config file.")
var namespace = flag.String("namespace", "", "Namespace name.")
var configMapName = flag.String("config-map-name", "", "Name of the ConfigMap to create.")

func createClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return cs
}

func createConfigFromFile() *v1.ConfigMap {
	f, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: *configMapName,
		},
		Data: map[string]string{
			path.Base(*configFile): string(f),
		},
	}
}

func main() {
	flag.Parse()
	config := createConfigFromFile()
	client := createClient().CoreV1().ConfigMaps(*namespace)
	_, err := client.Create(context.TODO(), config, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
}
