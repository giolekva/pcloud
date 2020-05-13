package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/golang/glog"
	"github.com/google/uuid"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	app "github.com/giolekva/pcloud/appmanager"
)

var kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file.")
var helmBin = flag.String("helm_bin", "/usr/local/bin/helm", "Path to the Helm binary.")
var port = flag.Int("port", 1234, "Port to listen on.")
var apiAddr = flag.String("api_addr", "", "PCloud API service address.")
var managerStoreFile = flag.String("manager_store_file", "", "Persistent file containing installed application information.")

var helmUploadPage = `
<html>
<head>
       <title>Upload Helm chart</title>
</head>
<body>
<form enctype="multipart/form-data" method="post">
    <input type="file" name="chartfile" />
    <input type="submit" value="upload" />
</form>
</body>
</html>
`

type handler struct {
	manager  *app.Manager
	nsClient corev1.NamespaceInterface
}

func (hn *handler) handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		_, err := io.WriteString(w, helmUploadPage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == "POST" {
		r.ParseMultipartForm(1000000)
		file, handler, err := r.FormFile("chartfile")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		tmp := uuid.New().String()
		if tmp == "" {
			http.Error(w, "Could not generate temp dir", http.StatusInternalServerError)
			return
		}
		p := "/tmp/" + tmp
		// TODO(giolekva): defer rmdir
		if err := syscall.Mkdir(p, 0777); err != nil {
			http.Error(w, "Could not create temp dir", http.StatusInternalServerError)
			return
		}
		p += "/" + handler.Filename
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		_, err = io.Copy(f, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = hn.installHelmChart(p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Installed"))
	}
}

type trigger struct {
	Namespace string `json:"namespace"`
	Template  string `json:"template"`
}

func (hn *handler) handleTriggers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Only GET method is supported on /triggers", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// TODO(giolekva): check if exists
	triggerOnType := r.Form["trigger_on_type"][0]
	triggerOnEvent := r.Form["trigger_on_event"][0]
	var triggers []trigger
	for _, a := range hn.manager.Apps {
		if a.Triggers == nil {
			continue
		}
		for _, t := range a.Triggers.Triggers {
			if t.TriggerOn.Type == triggerOnType && t.TriggerOn.Event == triggerOnEvent {
				triggers = append(triggers, trigger{a.Namespace, t.Template})
			}
		}
	}
	respBody, err := json.Marshal(triggers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, bytes.NewReader(respBody)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (hn *handler) installHelmChart(path string) error {
	h, err := app.HelmChartFromTar(path)
	if err != nil {
		return err
	}
	if err = app.InstallSchema(h.Schema, *apiAddr); err != nil {
		return err
	}
	glog.Infof("Installed schema: %s", h.Schema)
	namespace := fmt.Sprintf("app-%s", h.Name)
	if err = createNamespace(hn.nsClient, namespace); err != nil {
		return err
	}
	glog.Infof("Created namespaces: %s", namespace)
	if err = h.Install(
		*helmBin,
		map[string]string{}); err != nil {
		return err
	}
	glog.Info("Deployed")
	hn.manager.Apps[h.Name] = app.App{namespace, h.Triggers}
	app.StoreManagerStateToFile(hn.manager, *managerStoreFile)
	glog.Info("Installed")
	return nil
}

func createNamespace(nsClient corev1.NamespaceInterface, name string) error {
	_, err := nsClient.Create(
		context.TODO(),
		&apiv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name}},
		metav1.CreateOptions{})
	return err
}

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
	namespaces := clientset.CoreV1().Namespaces()
	manager, err := app.LoadManagerStateFromFile(*managerStoreFile)
	if err != nil {
		glog.Fatalf("Could ot initialize manager: %v", err)
	}
	h := handler{manager, namespaces}
	http.HandleFunc("/triggers", h.handleTriggers)
	http.HandleFunc("/", h.handleInstall)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

}
