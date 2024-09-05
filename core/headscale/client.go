package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var ErrorAlreadyExists = errors.New("already exists")

type client struct {
	config string
}

func newClient(config string) *client {
	return &client{
		config: fmt.Sprintf("--config=%s", config),
	}
}

func (c *client) createUser(name string) error {
	cmd := exec.Command("headscale", c.config, "users", "create", name)
	out, err := cmd.Output()
	outStr := string(out)
	if err != nil && strings.Contains(outStr, "User already exists") {
		return ErrorAlreadyExists
	}
	return err
}

func (c *client) createPreAuthKey(user string) (string, error) {
	// TODO(giolekva): make expiration configurable, and auto-refresh
	cmd := exec.Command("headscale", c.config, "--user", user, "preauthkeys", "create", "--reusable", "--expiration", "365d")
	out, err := cmd.Output()
	fmt.Println(string(out))
	if err != nil {
		return "", err
	}
	return extractLastLine(string(out))
}

func (c *client) expirePreAuthKey(user, authKey string) error {
	cmd := exec.Command("headscale", c.config, "--user", user, "preauthkeys", "expire", authKey)
	out, err := cmd.Output()
	fmt.Println(string(out))
	if err != nil {
		return err
	}
	return nil
}

func (c *client) expireUserNode(user, node string) error {
	id, err := c.getNodeId(user, node)
	if err != nil {
		return err
	}
	cmd := exec.Command("headscale", c.config, "node", "expire", "--identifier", id)
	out, err := cmd.Output()
	fmt.Println(string(out))
	if err != nil {
		return err
	}
	return nil
}

func (c *client) removeUserNode(user, node string) error {
	id, err := c.getNodeId(user, node)
	if err != nil {
		return err
	}
	cmd := exec.Command("headscale", c.config, "node", "delete", "--identifier", id, "--force")
	out, err := cmd.Output()
	fmt.Println(string(out))
	if err != nil {
		return err
	}
	return nil
}

func (c *client) enableRoute(id string) error {
	cmd := exec.Command("headscale", c.config, "routes", "enable", "-r", id)
	out, err := cmd.Output()
	fmt.Println(string(out))
	return err
}

type nodeInfo struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (c *client) getNodeId(user, node string) (string, error) {
	cmd := exec.Command("headscale", c.config, "--user", user, "node", "list", "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var nodes []nodeInfo
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&nodes); err != nil {
		return "", err
	}
	for _, n := range nodes {
		if n.Name == node {
			return strconv.Itoa(n.Id), nil
		}
	}
	return "", fmt.Errorf("not found")
}

func extractLastLine(s string) (string, error) {
	items := strings.Split(s, "\n")
	for i := len(items) - 1; i >= 0; i-- {
		t := strings.TrimSpace(items[i])
		if t != "" {
			return t, nil
		}
	}
	return "", fmt.Errorf("All lines are empty")
}
