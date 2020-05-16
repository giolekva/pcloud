package appmanager

import (
	"bytes"
	"context"
	"text/template"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
)

var leftDelim = "{-{"
var rightDelim = "}-}"

type Launcher interface {
	Launch(ns, tmpl string, args map[string]interface{}) error
}

type k8sLauncher struct {
	client *kubernetes.Clientset
}

func NewK8sLauncher(client *kubernetes.Clientset) Launcher {
	return &k8sLauncher{client}
}

func (k *k8sLauncher) Launch(ns, tmpl string, args map[string]interface{}) error {
	pod, err := renderTemplate(tmpl, args)
	if err != nil {
		return err
	}
	pods := k.client.CoreV1().Pods(ns)
	resp, err := pods.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	glog.Infof("Pod created: %s", resp)
	return nil
}

func renderTemplate(tmpl string, args map[string]interface{}) (*apiv1.Pod, error) {
	t, err := template.New("action").Delims(leftDelim, rightDelim).Parse(tmpl)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, args); err != nil {
		return nil, err
	}
	var pod apiv1.Pod
	dec := yaml.NewYAMLOrJSONDecoder(&b, 100)
	if err := dec.Decode(&pod); err != nil {
		return nil, err
	}
	return &pod, nil
}
