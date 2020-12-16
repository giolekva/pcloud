package events

import (
	"time"

	"k8s.io/client-go/kubernetes"

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
				err := p.appManager.LaunchAction(t.App, t.Action, args{event.NodeId, p.pcloudApi, p.objectStoreApi})
				// TODO(giolekva): do not simply ignore error and monitor progress
				if err != nil {
					continue
				}
				glog.Info("Launched action: %s %s", t.App, t.Action)
			}
			p.store.MarkEventDone(event)
		}
	}
}

type args struct {
	Id              string
	PCloudApiAddr   string
	ObjectStoreAddr string
}
