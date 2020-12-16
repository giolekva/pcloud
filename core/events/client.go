package events

import (
	"bytes"
	"encoding/json"
	// "errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

var jsonContentType = "application/json"

type query struct {
	Query string `json:"query"`
}

var getAllNewImageEventsTmpl = `{
  queryImageEvent(filter: {
    state: {
      eq: NEW
      le: NEW
      lt: NEW
      ge: NEW
      gt: NEW
    }
  }) {
    id
    node {
      id
    }
  }
}`

var markEventDoneTmpl = `mutation {
  updateImageEvent(input: {
    filter: {
      id: ["%s"]
    },
    set: {
      state: DONE
    }
  }) {
    numUids
  }
}`

// Implements EventStore
type GraphQLClient struct {
	apiAddr string
}

func NewGraphQLClient(apiAddr string) EventStore {
	return &GraphQLClient{apiAddr}
}

type location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type gqlError struct {
	Message   string     `json:"message"`
	Locations []location `json:"location"`
}

type gqlNode struct {
	Id string `json:"id"`
}

type gqlEvent struct {
	Id    string  `json:"id"`
	State string  `json:"state"`
	Node  gqlNode `json:"node"`
}

type gqlData struct {
	Events []gqlEvent `json:"queryImageEvent"`
}

type queryResp struct {
	Errors []gqlError `json:"errors"`
	Data   gqlData    `json:"data"`
}

func (c *GraphQLClient) GetEventsInState(state EventState) ([]Event, error) {
	q := query{getAllNewImageEventsTmpl}
	qJson, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(c.apiAddr, jsonContentType, bytes.NewReader(qJson))
	if err != nil {
		return nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	glog.Info(string(respBody))
	var gqlResp gqlData
	err = json.Unmarshal(respBody, &gqlResp)
	if err != nil {
		return nil, err
	}
	// if len(gqlResp.Errors) != 0 {
	// 	return nil, errors.New(fmt.Sprintf("%v", gqlResp.Errors))
	// }
	var events []Event
	for _, e := range gqlResp.Events {
		events = append(events, Event{e.Id, EventStateNew, e.Node.Id})
	}
	return events, nil
}

func (c *GraphQLClient) MarkEventDone(event Event) error {
	q := query{fmt.Sprintf(markEventDoneTmpl, event.Id)}
	qJson, err := json.Marshal(q)
	if err != nil {
		return err
	}
	_, err = http.Post(c.apiAddr, jsonContentType, bytes.NewReader(qJson))
	if err != nil {
		return err
	}
	// TODO(giolekva): check errors field
	return nil
}
