package controllers

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
)

const zoneConfigFilename = "coredns.conf"
const rootConfigFilename = "coredns.conf"
const importAllConfigFiles = "import */" + zoneConfigFilename

type ZoneStore interface {
	ConfigPath() string
	AddDNSSec(key DNSSecKey) error
}

type ZoneConfig struct {
	Zone        string
	PublicIPs   []string
	PrivateIP   string
	Nameservers []string
	DNSSec      *DNSSecKey
}

type ZoneStoreFactory interface {
	ConfigPath() string
	Create(zone ZoneConfig) (ZoneStore, error)
	Debug()
}

type fsZoneStoreFactory struct {
	fs billy.Filesystem
}

func NewFSZoneStoreFactory(fs billy.Filesystem) (ZoneStoreFactory, error) {
	if err := util.WriteFile(fs, rootConfigFilename, []byte(importAllConfigFiles), os.ModePerm); err != nil {
		return nil, err
	}
	return &fsZoneStoreFactory{fs}, nil
}

func (f *fsZoneStoreFactory) ConfigPath() string {
	return f.fs.Join(f.fs.Root(), rootConfigFilename)
}

func (f *fsZoneStoreFactory) Debug() {
	fmt.Println("------------")
	util.Walk(f.fs, ".", func(path string, info fs.FileInfo, err error) error {
		fmt.Println(path)
		if !info.IsDir() {
			r, err := f.fs.Open(path)
			if err != nil {
				return err
			}
			defer r.Close()
			_, err = io.Copy(os.Stdout, r)
			return err
		}
		return nil
	})
	fmt.Println("++++++++++++++")
}

func (f *fsZoneStoreFactory) Create(zone ZoneConfig) (ZoneStore, error) {
	if err := f.fs.MkdirAll(zone.Zone, fs.ModePerm); err != nil {
		return nil, err
	}
	zfs, err := f.fs.Chroot(zone.Zone)
	if err != nil {
		return nil, err
	}
	z, err := NewFSZoneStore(zone, zfs)
	if err != nil {
		defer func() {
			if err := f.fs.Remove(zone.Zone); err != nil {
				fmt.Printf("Failed to remove zone directory: %s\n", err.Error())
			}
		}()
	}
	return z, nil
}

type fsZoneStore struct {
	zone ZoneConfig
	fs   billy.Filesystem
}

func NewFSZoneStore(zone ZoneConfig, fs billy.Filesystem) (ZoneStore, error) {
	if zone.DNSSec != nil {
		sec := zone.DNSSec
		if err := util.WriteFile(fs, sec.Basename+".key", sec.Key, 0644); err != nil {
			return nil, err
		}
		if err := util.WriteFile(fs, sec.Basename+".private", sec.Private, 0600); err != nil {
			return nil, err
		}
	}
	conf, err := fs.Create(zoneConfigFilename)
	if err != nil {
		return nil, err
	}
	defer conf.Close()
	configTmpl, err := template.New("config").Funcs(sprig.TxtFuncMap()).Parse(`
{{ .zone.Zone }}:53 {
	file {{ .rootDir }}/zone.db
	errors
    {{ if .zone.DNSSec }}
	dnssec {
		key file {{ .rootDir}}/{{ .zone.DNSSec.Basename }}
	}
    {{ end }}
	log
	health {
		lameduck 5s
	}
	ready
	cache 30
	loop
	reload
	loadbalance
}`)
	if err != nil {
		return nil, err
	}
	if err := configTmpl.Execute(conf, map[string]any{
		"zone":    zone,
		"rootDir": fs.Root(),
	}); err != nil {
		return nil, err
	}
	recordsTmpl, err := template.New("records").Funcs(sprig.TxtFuncMap()).Parse(`
{{ .zone }}.   IN SOA ns1.{{ .zone }}. hostmaster.{{ .zone }}. 2015082541 7200 3600 1209600 3600
{{ range $i, $ns := .nameservers }}
ns{{ add1 $i }} 10800 IN A {{ $ns }}
{{ end }}
{{ range .publicIngressIPs }}
@ 10800 IN A {{ . }}
{{ end }}
* 10800 IN CNAME {{ .zone }}.
p 10800 IN CNAME {{ .zone }}.
*.p 10800 IN A {{ .privateIngressIP }}
`)
	records, err := fs.Create("zone.db")
	if err != nil {
		return nil, err
	}
	defer records.Close()
	if err := recordsTmpl.Execute(records, map[string]any{
		"zone":             zone.Zone,
		"publicIngressIPs": zone.PublicIPs,
		"privateIngressIP": zone.PrivateIP,
		"nameservers":      zone.Nameservers,
	}); err != nil {
		return nil, err
	}
	return &fsZoneStore{zone, fs}, nil
}

func (s *fsZoneStore) ConfigPath() string {
	return s.fs.Join(s.fs.Root(), zoneConfigFilename)
}

func (s *fsZoneStore) AddDNSSec(key DNSSecKey) error {
	return nil
}
