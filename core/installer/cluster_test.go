package installer

import (
	"net"
	"strings"
	"testing"
)

func TestParseNginxProxyConfig(t *testing.T) {
	cfg, err := ParseNginxProxyConfig(strings.NewReader(`nginx.conf: |
# user       www www;
worker_processes  1;
error_log   /dev/null   crit;
# pid        logs/nginx.pid;
worker_rlimit_nofile 8192;
events {
	worker_connections  1024;
}
http {
	error_log /var/log/nginx/error.log debug;
	log_format dodo '$http_host $proxy_host $status';
	access_log /var/log/nginx/access.log dodo;
	map $http_host $backend {
		a A;
		b B;
	}
	server {
		listen 9090;
		location / {
			resolver 1.1.1.1;
			resolver 2.2.2.2;
			proxy_pass http://$backend;
		}
	}
}
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9090 {
		t.Errorf("invalid port: expected 9090, got %d", cfg.Port)
	}
	if len(cfg.Resolvers) != 2 ||
		!cfg.Resolvers[0].Equal(net.ParseIP("1.1.1.1")) ||
		!cfg.Resolvers[1].Equal(net.ParseIP("2.2.2.2")) {
		t.Errorf("invalid resolvers: expected [1.1.1.1 2.2.2.2], got %s", cfg.Resolvers)
	}
	if len(cfg.Proxies) != 2 ||
		cfg.Proxies["a"] != "A" ||
		cfg.Proxies["b"] != "B" {
		t.Errorf("invalid proxies: expected map[a:A, b:B], got %s", cfg.Proxies)
	}
}

func TestRenderNginxProxyConfig(t *testing.T) {
	cfg := NginxProxyConfig{
		Port:      8080,
		Resolvers: []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2")},
		Proxies: map[string]string{
			"a": "A",
			"b": "B",
		},
		PreConf: []string{"line1", "line2"},
	}
	var buf strings.Builder
	if err := cfg.Render(&buf); err != nil {
		t.Fatal(err)
	}
	t.Log(buf.String())
}
