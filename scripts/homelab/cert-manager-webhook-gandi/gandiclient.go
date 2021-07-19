package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"
)

const (
	GandiLiveDnsBaseUrl = "https://dns.api.gandi.net/api/v5"
)

type GandiClient struct {
	apiKey              string
	dumpRequestResponse bool
}

type GandiRRSet struct {
	Type   string   `json:"rrset_type"`
	TTL    int      `json:"rrset_ttl"`
	Name   string   `json:"rrset_name"`
	Values []string `json:"rrset_values"`
}

type GandiRRSetValues struct {
	TTL    int      `json:"rrset_ttl"`
	Values []string `json:"rrset_values"`
}

func NewGandiClient(apiKey string) *GandiClient {
	return &GandiClient{
		apiKey: apiKey,
		dumpRequestResponse: false,
	}
}

func (c *GandiClient) gandiRecordsUrl(domain string) string {
	return fmt.Sprintf("%s/domains/%s/records", GandiLiveDnsBaseUrl, domain)
}

func (c *GandiClient) doRequest(req *http.Request, readResponseBody bool) (int, []byte, error) {
	if c.dumpRequestResponse {
		dump, _ := httputil.DumpRequest(req, true)
		fmt.Printf("Request: %q\n", dump)
	}

	req.Header.Set("X-Api-Key", c.apiKey)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	if c.dumpRequestResponse {
		dump, _ := httputil.DumpResponse(res, true)
		fmt.Printf("Response: %q\n", dump)
	}

	if res.StatusCode == http.StatusOK && readResponseBody {
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return 0, nil, err
		}
		return res.StatusCode, data, nil
	}

	return res.StatusCode, nil, nil
}

func (c *GandiClient) HasTxtRecord(domain *string, name *string) (bool, error) {
	// curl -H "X-Api-Key: $APIKEY" \
	//     https://dns.api.gandi.net/api/v5/domains/<DOMAIN>/records/<NAME>/<TYPE>
	url := fmt.Sprintf("%s/%s/TXT", c.gandiRecordsUrl(*domain), *name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	status, _, err := c.doRequest(req, false)
	if err != nil {
		return false, err
	}

	if status == http.StatusNotFound {
		return false, nil
	} else if status == http.StatusOK {
		// Maybe parse response body here to really ensure that the record is present
		return true, nil
	} else {
		return false, fmt.Errorf("unexpected HTTP status: %d", status)
	}
}

func (c *GandiClient) CreateTxtRecord(domain *string, name *string, value *string, ttl int) error {
	// curl -X POST -H "Content-Type: application/json" \
	//             -H "X-Api-Key: $APIKEY" \
	//             -d '{"rrset_name": "<NAME>",
	//                  "rrset_type": "<TYPE>",
	//                  "rrset_ttl": 10800,
	//                  "rrset_values": ["<VALUE>"]}' \
	//             https://dns.api.gandi.net/api/v5/domains/<DOMAIN>/records
	rrs := GandiRRSet{Name: *name, Type: "TXT", TTL: ttl, Values: []string{*value}}
	body, err := json.Marshal(rrs)
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	url := c.gandiRecordsUrl(*domain)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	status, _, err := c.doRequest(req, false)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed creating TXT record: %v", err)
	}

	return nil
}

func (c *GandiClient) UpdateTxtRecord(domain *string, name *string, value *string, ttl int) error {
	// curl -X PUT -H "Content-Type: application/json" \
	//            -H "X-Api-Key: $APIKEY" \
	//            -d '{"rrset_ttl": 10800,
	//                 "rrset_values":["<VALUE>"]}' \
	//            https://dns.api.gandi.net/api/v5/domains/<DOMAIN>/records/<NAME>/<TYPE>
	rrs := GandiRRSetValues{TTL: ttl, Values: []string{*value}}
	body, err := json.Marshal(rrs)
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	url := fmt.Sprintf("%s/%s/TXT", c.gandiRecordsUrl(*domain), *name)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	status, _, err := c.doRequest(req, false)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed updating TXT record: %v", err)
	}

	return nil
}

func (c *GandiClient) DeleteTxtRecord(domain *string, name *string) error {
	// curl -X DELETE -H "Content-Type: application/json" \
	//  -H "X-Api-Key: $APIKEY" \
	// https://dns.api.gandi.net/api/v5/domains/<DOMAIN>/records/<NAME>/<TYPE>
	url := fmt.Sprintf("%s/%s/TXT", c.gandiRecordsUrl(*domain), *name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	status, _, err := c.doRequest(req, false)
	if err != nil {
		return err
	}

	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("failed deleting TXT record: %v", err)
	}

	return nil
}

