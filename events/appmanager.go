package events

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Trigger struct {
	Namespace string `json:"namespace"`
	Template  string `json:"template"`
}

type AppManager interface {
	QueryTriggers(triggerOnType string, triggerOnEvent string) ([]Trigger, error)
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
