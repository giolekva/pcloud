package installer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type VPNAuthKeyGenerator interface {
	Generate(username string) (string, error)
}

type headscaleAPIClient struct {
	apiAddr string
}

func NewHeadscaleAPIClient(apiAddr string) VPNAuthKeyGenerator {
	return &headscaleAPIClient{apiAddr}
}

func (g *headscaleAPIClient) Generate(username string) (string, error) {
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
