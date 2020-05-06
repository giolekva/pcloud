package events

import (
	"context"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/golang/glog"
	// "github.com/itaysk/regogo"
)

type Processor interface {
	Start()
}

// Implements processor
type singleEventAtATimeProcessor struct {
	store     EventStore
	pods      corev1.PodInterface
	pcloudApi string
	// TODO(giolekva): Nodes themselves should be associated with object store
	objectStoreApi string
}

func NewSingleEventAtATimeProcessor(
	store EventStore,
	pods corev1.PodInterface,
	pcloudApi, objectStoreApi string) Processor {
	return &singleEventAtATimeProcessor{store, pods, pcloudApi, objectStoreApi}
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
			pod := createPod(event.NodeId, p.pcloudApi, p.objectStoreApi)
			glog.Info("Creating pod...")
			resp, err := p.pods.Create(context.TODO(), pod, metav1.CreateOptions{})
			if err != nil {
				glog.Error(err)
				continue
			}
			glog.Infof("Pod created: %s", resp)
			// TODO(giolekva): do not ignore error
			_ = monitorPod(resp, p.pods)
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

func createPod(id string, pcloudApi string, objectStoreApi string) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("event-%s", id)},
		Spec: apiv1.PodSpec{
			RestartPolicy: apiv1.RestartPolicyNever,
			Containers: []apiv1.Container{{
				Name:            "event",
				Image:           "giolekva/face-detector:latest",
				ImagePullPolicy: apiv1.PullAlways,
				Command:         []string{"python3", "main.py"},
				Args:            []string{pcloudApi, objectStoreApi, id}}}}}
}
