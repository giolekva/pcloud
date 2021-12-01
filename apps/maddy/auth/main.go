package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

var config = flag.String("config", "/etc/maddy/config/smtp-servers.conf", "Path to the configuration file with downstream SMTP server addresses per line.")

func readConfig(path string) ([]string, error) {
	inp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer inp.Close()
	lines := bufio.NewScanner(inp)
	ret := make([]string, 0)
	for lines.Scan() {
		ret = append(ret, lines.Text())
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	return ret, nil
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
	flag.Parse()
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
	smtpServers, err := readConfig(*config)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
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
