package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

var smtpServers = []string{
	"maddy.app-maddy.svc.cluster.local:587",
	"maddy.shveli-app-maddy.svc.cluster.local:587",
}

func auth(server, username, password string) (bool, error) {
	c, err := smtp.Dial(server)
	if err != nil {
		return false, err
	}
	if err := c.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		return false, err
	}
	if err := c.Auth(sasl.NewPlainClient(username, username, password)); err != nil {
		return false, err
	}
	return true, nil
}

func main() {
	inp := bufio.NewReader(os.Stdin)
	username, err := inp.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not read username")
		os.Exit(2)
	}
	username = username[:len(username)-1]
	password, err := inp.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not read password")
		os.Exit(2)
	}
	password = password[:len(password)-1]
	for _, s := range smtpServers {
		if ok, _ := auth(s, username, password); ok {
			os.Exit(0)
			// } else if err != nil {
			// 	fmt.Println(os.Stderr, err.Error())
			// 	os.Exit(2)
		}
	}
	os.Exit(1)
}
