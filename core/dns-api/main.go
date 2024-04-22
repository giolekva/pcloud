package main

import (
	"flag"
	"strings"
)

var port = flag.Int("port", 8080, "Port to listen on")
var rootDir = flag.String("root-dir", "", "Path to generate DNS sec keys")
var config = flag.String("config", "", "Coredns config file name")
var db = flag.String("db", "", "DNS records db file name")
var zone = flag.String("zone", "", "Zone domain")
var publicIPs = flag.String("public-ip", "", "Comma separated list of public IPs of the pcloud environment")
var privateIP = flag.String("private-ip", "", "Private IP of the pcloud environment")
var nameserverIPs = flag.String("nameserver-ip", "", "Comma separated list of nameserver IPs")

func main() {
	flag.Parse()
	publicIP := strings.Split(*publicIPs, ",")
	nameserverIP := strings.Split(*nameserverIPs, ",")
	fs := osFS{*rootDir}
	store, ds, err := NewStore(fs, *config, *db, *zone, publicIP, *privateIP, nameserverIP)
	if err != nil {
		panic(err)
	}
	server := NewServer(*port, *zone, ds, store, nameserverIP)
	server.Start()
}
