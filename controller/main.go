package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giolekva/pcloud/controller/schema"

	"github.com/golang/glog"
	"github.com/itaysk/regogo"
)

var kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file.")

var port = flag.Int("port", 123, "Port to listen on.")
var dgraphGqlAddress = flag.String("graphql_address", "", "GraphQL server address.")
var dgraphSchemaAddress = flag.String("dgraph_admin_address", "", "Dgraph server admin address.")

const imgJson = `{ objectPath: \"%s\"}`
const insertQuery = `mutation { add%s(input: [%s]) { %s { id } } }`
const getQuery = `{ "query": "{ get%s(id: \"%s\") { id objectPath } } " }`

type MinioWebhook struct {
	gql  schema.GraphQLClient
	pods corev1.PodInterface
}

func (m *MinioWebhook) minioHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not read HTTP request body", http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		return
	}
	glog.Infof("Received event from Minio: %s", string(body))
	key, err := regogo.Get(string(body), "input.Key")
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not find object key", http.StatusBadRequest)
		return
	}
	resp, err := m.gql.RunQuery(fmt.Sprintf(
		"mutation { addImage(input: [{objectPath: \"%s\"}]) { image { id }} }",
		key.String()))
	if err != nil {
		glog.Error(err)
		http.Error(w, "Can not add given objects", http.StatusInternalServerError)
		return
	}
	id, err := regogo.Get(resp, "input.addImage.image[0].id")
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not extract node id", http.StatusInternalServerError)
		return
	}
	glog.Infof("New image id: %s", id.String())
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("detect-faces-%s", id.String())},
		Spec: apiv1.PodSpec{
			RestartPolicy: apiv1.RestartPolicyNever,
			Containers: []apiv1.Container{{
				Name:            "detect-faces",
				Image:           "face-detector:latest",
				ImagePullPolicy: apiv1.PullNever,
				Command:         []string{"python", "main.py"},
				Args:            []string{"http://pcloud-controller-service.pcloud.svc:1111/graphql", "http://minio-hl-svc.minio.svc:9000", id.String()}}}}}
	glog.Info("Creating pod...")
	result, err := m.pods.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not start face detector", http.StatusInternalServerError)
		return
	}
	glog.Infof("Created deployment %q.\n", result.GetObjectMeta().GetName())
}

func (m *MinioWebhook) graphqlHandler(w http.ResponseWriter, r *http.Request) {
	glog.Infof("New GraphQL query received: %s", r.Method)
	err := r.ParseForm()
	if err != nil {
		glog.Error(err)
		http.Error(w, "Could not read query", http.StatusInternalServerError)
		return
	}
	query, ok := r.Form["query"]
	if !ok || len(query) != 1 {
		http.Error(w, "Exactly ouery parameter must be provided", http.StatusBadRequest)
		return
	}
	resp, err := m.gql.RunQuery(query[0])
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, resp)
	w.Header().Set("Content-Type", "application/json")
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
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	pods := clientset.CoreV1().Pods("pcloud")

	gqlClient, err := schema.NewDgraphClient(
		*dgraphGqlAddress, *dgraphSchemaAddress)
	if err != nil {
		panic(err)
	}
	// err = gqlClient.SetSchema(`
	// type Image {
	//      id: ID!
	//      objectPath: String! @search(by: [exact])
	// }

	// type ImageSegment {
	//      id: ID!
	//      upperLeftX: Int!
	//      upperLeftY: Int!
	//      lowerRightX: Int!
	//      lowerRightY: Int!
	//      sourceImage: Image!
	//      objectPath: String
	// }

	// extend type Image {
	//      segments: [ImageSegment] @hasInverse(field: sourceImage)
	// }`)
	// if err != nil {
	// 	panic(err)
	// }
	mw := MinioWebhook{gqlClient, pods}
	http.HandleFunc("/minio_webhook", mw.minioHandler)
	http.HandleFunc("/graphql", mw.graphqlHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
