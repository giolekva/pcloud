package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	nebulav1 "github.com/giolekva/pcloud/core/nebula/controller/apis/nebula/v1"
	clientset "github.com/giolekva/pcloud/core/nebula/controller/generated/clientset/versioned"
)

var port = flag.Int("port", 8080, "Port to listen on.")
var kubeConfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
var masterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

//go:embed templates/*
var tmpls embed.FS

type Templates struct {
	Index *template.Template
}

func ParseTemplates(fs embed.FS) (*Templates, error) {
	index, err := template.ParseFS(fs, "templates/index.html")
	if err != nil {
		return nil, err
	}
	return &Templates{index}, nil
}

type nebulaCA struct {
	Name      string
	Namespace string
	Nodes     []nebulaNode
}

type nebulaNode struct {
	Name      string
	Namespace string
	IP        string
}

type Manager struct {
	kubeClient   kubernetes.Interface
	nebulaClient clientset.Interface
}

func (m *Manager) ListAll() ([]*nebulaCA, error) {
	ret := make([]*nebulaCA, 0)
	cas, err := m.nebulaClient.LekvaV1().NebulaCAs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ca := range cas.Items {
		ret = append(ret, &nebulaCA{
			Name:      ca.Name,
			Namespace: ca.Namespace,
			Nodes:     make([]nebulaNode, 0),
		})
	}
	nodes, err := m.nebulaClient.LekvaV1().NebulaNodes("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		for _, ca := range ret {
			if ca.Name == node.Spec.CAName {
				ca.Nodes = append(ca.Nodes, nebulaNode{
					Name:      node.Name,
					Namespace: node.Namespace,
					IP:        node.Spec.IPCidr,
				})
			}
		}
	}
	return ret, nil
}

func (m *Manager) createNode(namespace, name, caNamespace, caName, ipCidr, pubKey string) (string, string, error) {
	node := &nebulav1.NebulaNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nebulav1.NebulaNodeSpec{
			CAName:      caName,
			CANamespace: caNamespace,
			IPCidr:      ipCidr,
			PubKey:      pubKey,
			SecretName:  fmt.Sprintf("%s-cert", name),
		},
	}
	node, err := m.nebulaClient.LekvaV1().NebulaNodes(namespace).Create(context.TODO(), node, metav1.CreateOptions{})
	if err != nil {
		return "", "", err
	}
	return node.Namespace, node.Name, nil
}

func (m *Manager) getNodeCertQR(namespace, name string) ([]byte, error) {
	node, err := m.nebulaClient.LekvaV1().NebulaNodes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	secret, err := m.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), node.Spec.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data["host.png"], nil
}

func (m *Manager) getCACertQR(namespace, name string) ([]byte, error) {
	ca, err := m.nebulaClient.LekvaV1().NebulaCAs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	secret, err := m.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), ca.Spec.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data["ca.png"], nil
}

type Handler struct {
	mgr   Manager
	tmpls *Templates
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	cas, err := h.mgr.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.tmpls.Index.Execute(w, cas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) handleNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	qr, err := h.mgr.getNodeCertQR(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "img/png")
	w.Write(qr)
}

func (h *Handler) handleCA(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	qr, err := h.mgr.getCACertQR(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "img/png")
	w.Write(qr)
}

func (h *Handler) handleSignNode(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _, err := h.mgr.createNode(
		r.FormValue("node-namespace"),
		r.FormValue("node-name"),
		r.FormValue("ca-namespace"),
		r.FormValue("ca-name"),
		r.FormValue("ip-cidr"),
		r.FormValue("pub-key"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	flag.Parse()
	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	nebulaClient := clientset.NewForConfigOrDie(cfg)
	t, err := ParseTemplates(tmpls)
	if err != nil {
		log.Fatal(err)
	}
	mgr := Manager{
		kubeClient:   kubeClient,
		nebulaClient: nebulaClient,
	}
	handler := Handler{
		mgr:   mgr,
		tmpls: t,
	}
	r := mux.NewRouter()
	r.HandleFunc("/node/{namespace:[a-zA-z0-9-]+}/{name:[a-zA-z0-9-]+}", handler.handleNode)
	r.HandleFunc("/ca/{namespace:[a-zA-z0-9-]+}/{name:[a-zA-z0-9-]+}", handler.handleCA)
	r.HandleFunc("/sign-node", handler.handleSignNode)
	r.HandleFunc("/", handler.handleIndex)
	http.Handle("/", r)
	fmt.Printf("Starting HTTP server on port: %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
