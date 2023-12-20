package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/miracl/conflate"
)

var baseFile = flag.String("base", "", "Path to the homeserver.yaml config file.")
var mergeWith = flag.String("merge-with", "", "Name of the file from config to merge.")
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

func newConig(data []byte) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: *configMapName,
		},
		Data: map[string]string{
			path.Base(*baseFile): string(data),
		},
	}
}

func main() {
	flag.Parse()
	client := createClient().CoreV1().ConfigMaps(*namespace)
	conf := conflate.New()
	generated, err := ioutil.ReadFile(*baseFile)
	if err != nil {
		panic(err)
	}
	fmt.Printf("--- BASE:\n%s\n", string(generated))
	if err := conf.AddData(generated); err != nil {
		panic(err)
	}
	mergeWith, err := ioutil.ReadFile(*mergeWith)
	if err != nil {
		panic(err)
	}
	fmt.Printf("--- MERGE WITH:\n%s\n", string(mergeWith))
	if err := conf.AddData(mergeWith); err != nil {
		panic(err)
	}
	merged, err := conf.MarshalYAML()
	if err != nil {
		panic(err)
	}
	fmt.Printf("--- MERGED:\n%s\n", string(merged))
	config := newConig(merged)
	if _, err := client.Create(context.TODO(), config, metav1.CreateOptions{}); err != nil {
		panic(err)
	}
}
