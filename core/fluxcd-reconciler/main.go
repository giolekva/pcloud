package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var port = flag.Int("port", 8080, "Port to listen on")
var kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig file")

const reconcileAnnotation = "reconcile.fluxcd.io/requestedAt"
const reconcileAtLayout = time.RFC3339Nano

type Server struct {
	port   int
	client dynamic.Interface
}

func NewServer(port int, client dynamic.Interface) *Server {
	return &Server{port, client}
}

func (s *Server) Start() {
	r := mux.NewRouter()
	r.Path("/source/git/{namespace}/{name}/reconcile").Methods("GET").HandlerFunc(s.sourceGitReconcile)
	r.Path("/kustomization/{namespace}/{name}/reconcile").Methods("GET").HandlerFunc(s.kustomizationReconcile)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func getReconciledAt(obj *unstructured.Unstructured) (string, error) {
	status, ok := obj.Object["status"]
	if !ok {
		return "", fmt.Errorf("status not found")
	}
	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("status not map")
	}
	val, ok := statusMap["lastHandledReconcileAt"]
	if !ok {
		return "", fmt.Errorf("lastHandledReconcileAt not found in status")
	}
	valStr, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("lastHandledReconcileAt not string")
	}
	return valStr, nil
}

func reconcile(
	client dynamic.Interface,
	res schema.GroupVersionResource,
	namespace string,
	name string,
) error {
	unstr, err := client.Resource(res).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	timeNowTime := time.Now()
	annotations := unstr.GetAnnotations()
	annotations[reconcileAnnotation] = timeNowTime.Format(reconcileAtLayout)
	unstr.SetAnnotations(annotations)
	unstr, err = client.Resource(res).Namespace(namespace).Update(context.TODO(), unstr, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	for {
		unstr, err := client.Resource(res).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		reconciledAt, err := getReconciledAt(unstr)
		if err != nil {
			return err
		}
		reconciledAtTime, err := time.Parse(reconcileAtLayout, reconciledAt)
		if err != nil {
			return err
		}
		reconciledAtTime = reconciledAtTime.Add(3 * time.Hour)
		if reconciledAtTime.After(timeNowTime) {
			return nil
		}
	}
}

func (s *Server) sourceGitReconcile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace, ok := vars["namespace"]
	if !ok {
		http.Error(w, "namespace missing", http.StatusBadRequest)
		return
	}
	name, ok := vars["name"]
	if !ok {
		http.Error(w, "name missing", http.StatusBadRequest)
		return
	}
	res := schema.GroupVersionResource{
		Group:    "source.toolkit.fluxcd.io",
		Version:  "v1",
		Resource: "gitrepositories",
	}
	if err := reconcile(s.client, res, namespace, name); err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) kustomizationReconcile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace, ok := vars["namespace"]
	if !ok {
		http.Error(w, "namespace missing", http.StatusBadRequest)
		return
	}
	name, ok := vars["name"]
	if !ok {
		http.Error(w, "name missing", http.StatusBadRequest)
		return
	}
	res := schema.GroupVersionResource{
		Group:    "kustomize.toolkit.fluxcd.io",
		Version:  "v1",
		Resource: "kustomizations",
	}
	if err := reconcile(s.client, res, namespace, name); err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
}

func NewKubeClient(kubeconfig string) (dynamic.Interface, error) {
	if kubeconfig == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return dynamic.NewForConfig(config)
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return dynamic.NewForConfig(config)
	}
}

func main() {
	flag.Parse()
	client, err := NewKubeClient(*kubeconfig)
	if err != nil {
		log.Fatal(err)
	}
	NewServer(*port, client).Start()
}
