package main

import (
	"fmt"
	"os/exec"
)

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
	fmt.Println(string(out))
	return err
}

func (c *client) createPreAuthKey(user string) (string, error) {
	// TODO(giolekva): make expiration configurable, and auto-refresh
	cmd := exec.Command("headscale", c.config, "--user", user, "preauthkeys", "create", "--reusable", "--expiration", "365d")
	out, err := cmd.Output()
	fmt.Println(string(out))
	return string(out), err
}

func (c *client) enableRoute(id string) error {
	// TODO(giolekva): make expiration configurable, and auto-refresh
	cmd := exec.Command("headscale", c.config, "routes", "enable", "-r", id)
	out, err := cmd.Output()
	fmt.Println(string(out))
	return err
}
