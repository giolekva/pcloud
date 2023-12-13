package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	dnsv1 "github.com/giolekva/pcloud/core/ns-controller/api/v1"
)

type NamespaceCreator interface {
	Create(name string) error
}

type ZoneInfo struct {
	Zone    string
	Records string
}

type ZoneStatusFetcher interface {
	Fetch(namespace, name string) (error, bool, ZoneInfo)
}

type realNamespaceCreator struct {
	clientset *kubernetes.Clientset
}

func (n *realNamespaceCreator) Create(name string) error {
	_, err := n.clientset.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       " ",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

type realZoneStatusFetcher struct {
	clientset dynamic.Interface
}

func (f *realZoneStatusFetcher) Fetch(namespace, name string) (error, bool, ZoneInfo) {
	dnsZoneRes := schema.GroupVersionResource{Group: "dodo.cloud.dodo.cloud", Version: "v1", Resource: "dnszones"}
	zoneUnstr, err := f.clientset.Resource(dnsZoneRes).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	fmt.Printf("%+v %+v\n", zoneUnstr, err)
	if err != nil {
		return err, false, ZoneInfo{}
	}
	var contents bytes.Buffer
	if err := json.NewEncoder(&contents).Encode(zoneUnstr.Object); err != nil {
		return err, false, ZoneInfo{}
	}
	var zone dnsv1.DNSZone
	if err := json.NewDecoder(&contents).Decode(&zone); err != nil {
		return err, false, ZoneInfo{}
	}
	return nil, zone.Status.Ready, ZoneInfo{zone.Spec.Zone, zone.Status.RecordsToPublish}
}

func NewNamespaceCreator(kubeconfig string) (NamespaceCreator, error) {
	clientset, err := NewKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &realNamespaceCreator{clientset}, nil
}

func NewZoneStatusFetcher(kubeconfig string) (ZoneStatusFetcher, error) {
	if kubeconfig == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		client, err := dynamic.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return &realZoneStatusFetcher{client}, nil

	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		client, err := dynamic.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return &realZoneStatusFetcher{client}, nil
	}
}

func NewKubeConfig(kubeconfig string) (*kubernetes.Clientset, error) {
	if kubeconfig == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)

	} else {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}
}
