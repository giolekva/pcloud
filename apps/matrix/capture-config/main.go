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

	"github.com/miracl/conflate"
)

var configFile = flag.String("config", "", "Path to the homeserver.yaml config file.")
var configToMerge = flag.String("config-to-merge", "", "Name of the configmap to merge with generated one.")
var toMergeFilename = flag.String("to-merge-filename", "", "Name of the file from config to merge.")
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

func createConfig(data []byte) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: *configMapName,
		},
		Data: map[string]string{
			path.Base(*configFile): string(data),
		},
	}
}

func main() {
	flag.Parse()
	client := createClient().CoreV1().ConfigMaps(*namespace)
	conf := conflate.New()
	generated, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}
	if err := conf.AddData(generated); err != nil {
		panic(err)
	}
	toMerge, err := client.Get(context.TODO(), *configToMerge, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	if err := conf.AddData([]byte(toMerge.Data[*toMergeFilename])); err != nil {
		panic(err)
	}
	merged, err := conf.MarshalYAML()
	if err != nil {
		panic(err)
	}
	config := createConfig(merged)
	if _, err := client.Create(context.TODO(), config, metav1.CreateOptions{}); err != nil {
		panic(err)
	}
}
