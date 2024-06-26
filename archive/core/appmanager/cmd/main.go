package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

	app "github.com/giolekva/pcloud/core/appmanager"
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
	client   *kubernetes.Clientset
	manager  *app.Manager
	launcher app.Launcher
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
	App    string `json:"app"`
	Action string `json:"action"`
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
		for _, t := range a.Triggers.Triggers {
			if t.TriggerOn.Type == triggerOnType && t.TriggerOn.Event == triggerOnEvent {
				triggers = append(triggers, trigger{a.Name, t.Action})
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

type actionReq struct {
	App    string                 `json:"app"`
	Action string                 `json:"action"`
	Args   map[string]interface{} `json:"args"`
}

func (hn *handler) handleLaunchAction(w http.ResponseWriter, r *http.Request) {
	actionStr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var req actionReq
	if err := json.Unmarshal(actionStr, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := hn.launchAction(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (hn *handler) launchAction(req actionReq) error {
	for _, a := range hn.manager.Apps {
		if a.Name != req.App {
			continue
		}
		for _, action := range a.Actions.Actions {
			if action.Name == req.Action {
				return hn.launcher.Launch(a.Namespace, action.Template, req.Args)
			}
		}
	}
	return fmt.Errorf("Action not found: %s %s", req.App, req.Action)
}

func (hn *handler) installHelmChart(path string) error {
	h, err := app.HelmChartFromTar(path)
	if err != nil {
		return err
	}
	if err := h.Render(
		*helmBin,
		map[string]string{}); err != nil {
		return err
	}
	glog.Info("Rendered templates")
	if err = app.InstallSchema(h.Schema, *apiAddr); err != nil {
		return err
	}
	glog.Infof("Installed schema: %s", h.Schema)
	err = createNamespace(hn.client.CoreV1().Namespaces(), h.Namespace)
	if err != nil {
		return err
	}
	glog.Infof("Created namespaces: %s", h.Namespace)
	if h.Type == "application" {
		if err := h.Install(*helmBin); err != nil {
			return err
		}
		glog.Info("Deployed")
	} else {
		glog.Info("Skipping deployment as we got library chart.")
	}
	hn.manager.Apps[h.Name] = app.App{h.Name, h.Namespace, h.Triggers, h.Actions}
	app.StoreManagerStateToFile(hn.manager, *managerStoreFile)
	for _, a := range h.Init.PostInstall.CallAction {
		if err := hn.launchAction(actionReq{a.App, a.Action, a.Args}); err != nil {
			return err
		}
	}
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
	manager, err := app.LoadManagerStateFromFile(*managerStoreFile)
	if err != nil {
		glog.Fatalf("Could ot initialize manager: %v", err)
	}
	glog.Info(manager)
	h := handler{clientset, manager, app.NewK8sLauncher(clientset)}
	http.HandleFunc("/triggers", h.handleTriggers)
	http.HandleFunc("/launch_action", h.handleLaunchAction)
	http.HandleFunc("/", h.handleInstall)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

}
