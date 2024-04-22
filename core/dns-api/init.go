package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/miekg/dns"
)

const coreDNSConfigTmpl = `
{{ .zone }}:53 {
	file {{ .dbFile }} {
		reload 1s
	}
	errors
	{{- if .dnsSecBasename }}
	dnssec {
		key file {{ .dnsSecBasename}}
	}
	{{- end }}
	log
	health {
		lameduck 5s
	}
	ready
	cache 30
	loop
	reload
	loadbalance
}
`

const recordsDBTmpl = `
{{- $zone := .zone }}
{{ $zone }}.   IN SOA ns1.{{ $zone }}. hostmaster.{{ $zone }}. {{ .nowUnix }} 7200 3600 1209600 3600
{{- range $i, $ns := .nameserverIP }}
ns{{ add1 $i }}.{{ $zone }}. 10800 IN A {{ $ns }}
{{- end }}
{{- range .publicIP }}
{{ $zone }}. 10800 IN A {{ . }}
*.{{ $zone }}. 10800 IN A {{ . }}
*.*.{{ $zone }}. 10800 IN A {{ . }}
{{- end }}
*.p.{{ $zone }}. 10800 IN A {{ .privateIP }}
`

func NewStore(fs FS, config string, db string, zone string, publicIP []string, privateIP string, nameserverIP []string) (RecordStore, string, error) {
	dnsSec, err := getDNSSecKey(fs, zone)
	if err != nil {
		return nil, "", err
	}
	if err := fs.Write(dnsSec.Basename+".key", string(dnsSec.Key)); err != nil {
		return nil, "", err
	}
	if err := fs.Write(dnsSec.Basename+".private", string(dnsSec.Private)); err != nil {
		return nil, "", err
	}
	if err := executeTemplate(fs, config, coreDNSConfigTmpl, map[string]any{
		"zone":           zone,
		"dbFile":         fs.AbsolutePath(db),
		"dnsSecBasename": fs.AbsolutePath(dnsSec.Basename),
	}); err != nil {
		return nil, "", err
	}
	ok, err := fs.Exists(db)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		if err := executeTemplate(fs, db, recordsDBTmpl, map[string]any{
			"zone":         zone,
			"publicIP":     publicIP,
			"privateIP":    privateIP,
			"nameserverIP": nameserverIP,
			"nowUnix":      NowUnix(),
		}); err != nil {
			return nil, "", err
		}
	}
	return &fsRecordStore{zone, publicIP, fs, db}, string(dnsSec.DS), nil
}

func getDNSSecKey(fs FS, zone string) (DNSSecKey, error) {
	const configFile = "dns-sec-key.json"
	ok, err := fs.Exists(configFile)
	if err != nil {
		return DNSSecKey{}, err
	}
	if ok {
		d, err := fs.Read(configFile)
		if err != nil {
			return DNSSecKey{}, err
		}
		var k DNSSecKey
		if err := json.Unmarshal([]byte(d), &k); err != nil {
			return DNSSecKey{}, err
		}
		return k, nil
	}
	k, err := newDNSSecKey(zone)
	if err != nil {
		return DNSSecKey{}, err
	}
	d, err := json.MarshalIndent(k, "", "\t")
	if err != nil {
		return DNSSecKey{}, err
	}
	if err := fs.Write(configFile, string(d)); err != nil {
		return DNSSecKey{}, err
	}
	return k, nil
}

type DNSSecKey struct {
	Basename string `json:"basename,omitempty"`
	Key      []byte `json:"key,omitempty"`
	Private  []byte `json:"private,omitempty"`
	DS       []byte `json:"ds,omitempty"`
}

func newDNSSecKey(zone string) (DNSSecKey, error) {
	key := &dns.DNSKEY{
		Hdr:       dns.RR_Header{Name: dns.Fqdn(zone), Class: dns.ClassINET, Ttl: 3600, Rrtype: dns.TypeDNSKEY},
		Algorithm: dns.ECDSAP256SHA256, Flags: 257, Protocol: 3,
	}
	priv, err := key.Generate(256)
	if err != nil {
		return DNSSecKey{}, err
	}
	return DNSSecKey{
		Basename: fmt.Sprintf("K%s+%03d+%05d", key.Header().Name, key.Algorithm, key.KeyTag()),
		Key:      []byte(key.String()),
		Private:  []byte(key.PrivateKeyString(priv)),
		DS:       []byte(key.ToDS(dns.SHA256).String()),
	}, nil
}

// TODO(gio): not going to work in 15 years?
// TODO(gio): remove 10 *
func NowUnix() uint32 {
	return 10 * uint32(time.Now().Unix())
}

func executeTemplate(fs FS, path string, contents string, values map[string]any) error {
	tmpl, err := template.New("tmpl").Funcs(sprig.TxtFuncMap()).Parse(contents)
	if err != nil {
		return err
	}
	var d strings.Builder
	if err := tmpl.Execute(&d, values); err != nil {
		return err
	}
	return fs.Write(path, strings.TrimSpace(d.String())+"\n")
}
