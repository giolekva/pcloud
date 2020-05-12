package main

import (
	"context"
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
var port = flag.Int("port", 1234, "Port to listen on.")
var apiAddr = flag.String("api_addr", "", "PCloud API service address.")

var helmUploadPage = `
<html>
<head>
       <title>Upload Helm chart</title>
</head>
<body>
<form enctype="multipart/form-data" action="/" method="post">
    <input type="file" name="chartfile" />
    <input type="submit" value="upload" />
</form>
</body>
</html>
`

type handler struct {
	nsClient corev1.NamespaceInterface
}

func (hn *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		if err = installHelmChart(p, hn.nsClient); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Installed"))
	}
}

func installHelmChart(path string, nsClient corev1.NamespaceInterface) error {
	h, err := app.HelmChartFromTar(path)
	if err != nil {
		return err
	}
	if err = app.InstallSchema(h.Schema, *apiAddr); err != nil {
		return err
	}
	glog.Infof("Installed schema: %s", h.Schema)
	namespace := fmt.Sprintf("app-%s", h.Name)
	if err = createNamespace(nsClient, namespace); err != nil {
		return err
	}
	glog.Infof("Created namespaces: %s", namespace)
	if err = h.Install(
		"/usr/local/bin/helm",
		map[string]string{}); err != nil {
		return err
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
	namespaces := clientset.CoreV1().Namespaces()
	http.Handle("/", &handler{namespaces})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

}
