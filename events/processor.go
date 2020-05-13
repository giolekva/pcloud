package events

import (
	"bytes"
	"context"
	"text/template"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/golang/glog"
	// "github.com/itaysk/regogo"
)

type Processor interface {
	Start()
}

// Implements processor
type singleEventAtATimeProcessor struct {
	store      EventStore
	appManager AppManager
	kube       *kubernetes.Clientset
	pcloudApi  string
	// TODO(giolekva): Nodes themselves should be associated with object store
	objectStoreApi string
}

func NewSingleEventAtATimeProcessor(
	store EventStore,
	appManager AppManager,
	kube *kubernetes.Clientset,
	pcloudApi, objectStoreApi string) Processor {
	return &singleEventAtATimeProcessor{store, appManager, kube, pcloudApi, objectStoreApi}
}

func (p *singleEventAtATimeProcessor) Start() {
	for {
		select {
		case <-time.After(30 * time.Second):
			events, err := p.store.GetEventsInState(EventStateNew)
			if err != nil {
				glog.Error(err)
				continue
			}
			if len(events) == 0 {
				continue
			}
			event := events[0]
			triggers, err := p.appManager.QueryTriggers("Image", string(EventStateNew))
			if err != nil {
				glog.Error(err)
				continue
			}
			for _, t := range triggers {
				pod, err := renderTriggerTemplate(t, event.NodeId, p.pcloudApi, p.objectStoreApi)
				if err != nil {
					glog.Errorf("Could not render trigger: %v %v", err, t)
					continue
				}
				glog.Info("Creating pod: %v", pod)
				pods := p.kube.CoreV1().Pods(t.Namespace)
				resp, err := pods.Create(context.TODO(), pod, metav1.CreateOptions{})
				if err != nil {
					glog.Error(err)
					continue
				}
				glog.Infof("Pod created: %s", resp)
				// TODO(giolekva): do not ignore error
				_ = monitorPod(resp, pods)
			}
			p.store.MarkEventDone(event)
		}
	}
}

func isInTerminalState(pod *apiv1.Pod) bool {
	return pod.Status.Phase == apiv1.PodSucceeded ||
		pod.Status.Phase == apiv1.PodFailed
}

func monitorPod(pod *apiv1.Pod, pods corev1.PodInterface) error {
	w, err := pods.Watch(context.TODO(), metav1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return err
	}
	for {
		select {
		case events, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			p := events.Object.(*apiv1.Pod)
			glog.Infof("Pod status: %s", pod.Status.Phase)
			if isInTerminalState(p) {
				glog.Info("Pod is DONE")
				w.Stop()
			}
		}
	}
	return nil
}

type args struct {
	Id              string
	PCloudApiAddr   string
	ObjectStoreAddr string
}

func renderTriggerTemplate(t Trigger, id string, pcloudApi string, objectStoreApi string) (*apiv1.Pod, error) {
	tmpl, err := template.New("trigger").Parse(t.Template)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := tmpl.Execute(&b, args{id, pcloudApi, objectStoreApi}); err != nil {
		return nil, err
	}
	var pod apiv1.Pod
	dec := yaml.NewYAMLOrJSONDecoder(&b, 100)
	if err := dec.Decode(&pod); err != nil {
		return nil, err
	}
	return &pod, nil
}
