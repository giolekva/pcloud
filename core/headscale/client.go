package main

import (
	"errors"
	"fmt"
	"os/exec"
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
	if err != nil {
		return "", err
	}
	fmt.Println(string(out))
	return extractLastLine(string(out))
}

func (c *client) enableRoute(id string) error {
	// TODO(giolekva): make expiration configurable, and auto-refresh
	cmd := exec.Command("headscale", c.config, "routes", "enable", "-r", id)
	out, err := cmd.Output()
	fmt.Println(string(out))
	return err
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
