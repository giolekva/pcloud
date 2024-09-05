package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type VPNAPIClient interface {
	GenerateAuthKey(username string) (string, error)
	ExpireKey(username, key string) error
	ExpireNode(username, node string) error
	RemoveNode(username, node string) error
}

type headscaleAPIClient struct {
	c       *http.Client
	apiAddr string
}

func NewHeadscaleAPIClient(apiAddr string) VPNAPIClient {
	return &headscaleAPIClient{
		&http.Client{},
		apiAddr,
	}
}

func (g *headscaleAPIClient) GenerateAuthKey(username string) (string, error) {
	resp, err := http.Post(fmt.Sprintf("%s/user/%s/preauthkey", g.apiAddr, username), "application/json", nil)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(buf.String())
	}
	return buf.String(), nil
}

type expirePreAuthKeyReq struct {
	AuthKey string `json:"authKey"`
}

func (g *headscaleAPIClient) ExpireKey(username, key string) error {
	addr, err := url.Parse(fmt.Sprintf("%s/user/%s/preauthkey", g.apiAddr, username))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(expirePreAuthKeyReq{key}); err != nil {
		return err
	}
	resp, err := g.c.Do(&http.Request{
		URL:    addr,
		Method: http.MethodDelete,
		Body:   io.NopCloser(&buf),
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return errors.New(buf.String())
	}
	return nil
}

func (g *headscaleAPIClient) ExpireNode(username, node string) error {
	resp, err := g.c.Post(
		fmt.Sprintf("%s/user/%s/node/%s/expire", g.apiAddr, username, node),
		"text/plain",
		nil,
	)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return errors.New(buf.String())
	}
	return nil
}

func (g *headscaleAPIClient) RemoveNode(username, node string) error {
	addr, err := url.Parse(fmt.Sprintf("%s/user/%s/node/%s", g.apiAddr, username, node))
	if err != nil {
		return err
	}
	resp, err := g.c.Do(&http.Request{
		URL:    addr,
		Method: http.MethodDelete,
		Body:   nil,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		return errors.New(buf.String())
	}
	return nil
}
