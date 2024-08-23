package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/giolekva/pcloud/core/installer/kube"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type NamespaceCreator interface {
	Create(name string) error
}

type ZoneInfo struct {
	Zone    string
	Records string
}

type ZoneStatusFetcher interface {
	Fetch(addr string) (string, error)
}

type noOpNamespaceCreator struct{}

func (n *noOpNamespaceCreator) Create(name string) error {
	return nil
}

func NewNoOpNamespaceCreator() NamespaceCreator {
	return &noOpNamespaceCreator{}
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

// TODO(gio): take http client
type realZoneStatusFetcher struct{}

func (f *realZoneStatusFetcher) Fetch(addr string) (string, error) {
	fmt.Printf("--- %s\n", addr)
	resp, err := http.Get(addr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func NewNamespaceCreator(opts kube.KubeConfigOpts) (NamespaceCreator, error) {
	clientset, err := kube.NewKubeClient(opts)
	if err != nil {
		return nil, err
	}
	return &realNamespaceCreator{clientset}, nil
}

func NewZoneStatusFetcher(kubeconfig string) (ZoneStatusFetcher, error) {
	return &realZoneStatusFetcher{}, nil
}

type HelmReleaseMonitor interface {
	IsReleased(namespace, name string) (bool, error)
}

type realHelmReleaseMonitor struct {
	d dynamic.Interface
}

func (m *realHelmReleaseMonitor) IsReleased(namespace, name string) (bool, error) {
	ctx := context.Background()
	res, err := m.d.Resource(schema.GroupVersionResource{"helm.toolkit.fluxcd.io", "v2beta1", "helmreleases"}).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	b, err := res.MarshalJSON()
	if err != nil {
		return false, err
	}
	var hr helmRelease
	if err := json.Unmarshal(b, &hr); err != nil {
		return false, err
	}
	for _, c := range hr.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return true, nil
		}
	}
	return false, nil
}

func NewHelmReleaseMonitor(kubeconfig string) (HelmReleaseMonitor, error) {
	c, err := kube.NewKubeClient(kube.KubeConfigOpts{KubeConfigPath: kubeconfig})
	if err != nil {
		return nil, err
	}
	d := dynamic.New(c.RESTClient())
	return &realHelmReleaseMonitor{d}, nil
}
