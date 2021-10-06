package main

import (
	"flag"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	controllers "github.com/giolekva/pcloud/core/nebula/controllers"
	clientset "github.com/giolekva/pcloud/core/nebula/generated/clientset/versioned"
	"github.com/giolekva/pcloud/core/nebula/generated/clientset/versioned/scheme"
	informers "github.com/giolekva/pcloud/core/nebula/generated/informers/externalversions"
	nebulascheme "k8s.io/sample-controller/pkg/generated/clientset/versioned/scheme"
)

var kubeConfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
var masterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
var nebulaCert = flag.String("nebula-cert", "", "Path to the nebula-cert binary.")

func main() {
	flag.Parse()
	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeConfig)
	if err != nil {
		panic(err)
	}
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	nebulaClient := clientset.NewForConfigOrDie(cfg)
	utilruntime.Must(nebulascheme.AddToScheme(scheme.Scheme))
	nebulaInformerFactory := informers.NewSharedInformerFactory(nebulaClient, 5*time.Second)
	c := controllers.NewCAController(kc, nebulaClient, nebulaInformerFactory, *nebulaCert)
	stopCh := make(chan struct{})
	nebulaInformerFactory.Start(stopCh)
	if err := c.Run(1, stopCh); err != nil {
		panic(err)
	}
}
