package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

type Trigger struct {
	App    string `json:"app"`
	Action string `json:"action"`
}

type AppManager interface {
	QueryTriggers(triggerOnType, triggerOnEvent string) ([]Trigger, error)
	// TODO(giolekva): must return launched action id to enable monitoring
	LaunchAction(app, action string, args interface{}) error
}

type appManagerClient struct {
	addr string
}

func NewAppManagerClient(addr string) AppManager {
	return &appManagerClient{addr}
}

func (c *appManagerClient) QueryTriggers(triggerOnType string, triggerOnEvent string) ([]Trigger, error) {
	triggerUrl := fmt.Sprintf("%s/triggers?trigger_on_type=%s&trigger_on_event=%s",
		c.addr, triggerOnType, triggerOnEvent)
	resp, err := http.Get(triggerUrl)
	if err != nil {
		return nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	triggers := make([]Trigger, 0)
	if err := json.Unmarshal(respBody, &triggers); err != nil {
		return nil, err
	}
	return triggers, nil
}

type actionArgs struct {
	App    string      `json:"app"`
	Action string      `json:"action"`
	Args   interface{} `json:"args"`
}

func (c *appManagerClient) LaunchAction(app, action string, args interface{}) error {
	actionUrl := fmt.Sprintf("%s/launch_action", c.addr)
	reqJson, err := json.Marshal(actionArgs{app, action, args})
	if err != nil {
		return err
	}
	resp, err := http.Post(actionUrl, "application/json", bytes.NewReader(reqJson))
	if err != nil {
		return err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	glog.Info("Triggered action: %s", string(respBody))
	return nil
}
