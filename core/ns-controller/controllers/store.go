package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
)

const dodoConfigFilename = "dodo.json"
const zoneConfigFilename = "coredns.conf"
const rootConfigFilename = "coredns.conf"
const importAllConfigFiles = "import */" + zoneConfigFilename

type ZoneStore interface {
	ConfigPath() string
	CreateConfigFile() error
	AddDNSSec(key DNSSecKey) error
	AddTextRecord(entry, txt string) error
	DeleteTextRecord(entry, txt string) error
}

type ZoneConfig struct {
	Zone        string     `json:"zone,omitempty"`
	PublicIPs   []string   `json:"publicIPs,omitempty"`
	PrivateIP   string     `json:"privateIP,omitempty"`
	Nameservers []string   `json:"nameservers,omitempty"`
	DNSSec      *DNSSecKey `json:"dnsSec,omitempty"`
}

func GenerateNSRecords(z ZoneConfig) []string {
	subdomain := strings.Split(z.Zone, ",")[0]
	ret := make([]string, 0)
	for i, ip := range z.Nameservers {
		ret = append(ret, fmt.Sprintf("ns%d.%s 10800 IN A %s", i+1, z.Zone, ip))
		ret = append(ret, fmt.Sprintf("%s. 10800 IN NS ns%d.%s.", subdomain, i+1, z.Zone))
	}
	return ret
}

type ZoneStoreFactory interface {
	ConfigPath() string
	Create(zone ZoneConfig) (ZoneStore, error)
	Get(zone string) (ZoneStore, error)
	Debug()
	Purge()
}

type fsZoneStoreFactory struct {
	fs    billy.Filesystem
	zones map[string]ZoneStore
}

func NewFSZoneStoreFactory(fs billy.Filesystem) (ZoneStoreFactory, error) {
	if err := util.WriteFile(fs, rootConfigFilename, []byte(importAllConfigFiles), os.ModePerm); err != nil {
		return nil, err
	}
	f, err := fs.ReadDir(".")
	if err != nil {
		return nil, err
	}
	zf := fsZoneStoreFactory{fs: fs, zones: make(map[string]ZoneStore)}
	for _, i := range f {
		if i.IsDir() {
			var zone ZoneConfig
			r, err := fs.Open(fs.Join(i.Name(), dodoConfigFilename))
			if err != nil {
				continue // TODO(gio): clean up the dir to enforce config file
			}
			defer r.Close()
			if err := json.NewDecoder(r).Decode(&zone); err != nil {
				return nil, err
			}
			zfs, err := fs.Chroot(zone.Zone)
			if err != nil {
				return nil, err
			}
			z, err := NewFSZoneStore(zone, zfs)
			zf.zones[zone.Zone] = z
		}
	}
	return &zf, nil
}

func (f *fsZoneStoreFactory) ConfigPath() string {
	return f.fs.Join(f.fs.Root(), rootConfigFilename)
}

func (f *fsZoneStoreFactory) Purge() {
	items, _ := f.fs.ReadDir(".")
	for _, i := range items {
		f.fs.Remove(i.Name())
	}
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

func (f *fsZoneStoreFactory) Get(zone string) (ZoneStore, error) {
	if z, ok := f.zones[zone]; ok {
		return z, nil
	}
	return nil, fmt.Errorf("%s zone not found", zone)
}

func (f *fsZoneStoreFactory) Create(zone ZoneConfig) (ZoneStore, error) {
	if z, ok := f.zones[zone.Zone]; ok {
		return z, nil
	}
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
	f.zones[zone.Zone] = z
	return z, nil
}

type fsZoneStore struct {
	zone ZoneConfig
	fs   billy.Filesystem
}

func NewFSZoneStore(zone ZoneConfig, fs billy.Filesystem) (ZoneStore, error) {
	return &fsZoneStore{zone, fs}, nil
}

func (s *fsZoneStore) CreateConfigFile() error {
	{
		w, err := s.fs.Create(dodoConfigFilename)
		if err != nil {
			return err
		}
		defer w.Close()
		if err := json.NewEncoder(w).Encode(s.zone); err != nil {
			return err
		}
	}
	zone := s.zone
	fs := s.fs
	if zone.DNSSec != nil {
		sec := zone.DNSSec
		if err := util.WriteFile(fs, sec.Basename+".key", sec.Key, 0644); err != nil {
			return err
		}
		if err := util.WriteFile(fs, sec.Basename+".private", sec.Private, 0600); err != nil {
			return err
		}
	}
	conf, err := fs.Create(zoneConfigFilename)
	if err != nil {
		return err
	}
	defer conf.Close()
	configTmpl, err := template.New("config").Funcs(sprig.TxtFuncMap()).Parse(`
{{ .zone.Zone }}:53 {
	file {{ .rootDir }}/zone.db {
      reload 1s
    }
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
		return err
	}
	if err := configTmpl.Execute(conf, map[string]any{
		"zone":    zone,
		"rootDir": fs.Root(),
	}); err != nil {
		return err
	}
	recordsTmpl, err := template.New("records").Funcs(sprig.TxtFuncMap()).Parse(`
{{ $zone := .zone }}
{{ $zone }}.   IN SOA ns1.{{ $zone }}. hostmaster.{{ $zone }}. {{ .nowUnix }} 7200 3600 1209600 3600
{{ range $i, $ns := .nameservers }}
ns{{ add1 $i }}.{{ $zone }}. 10800 IN A {{ $ns }}
{{ end }}
{{ range .publicIngressIPs }}
{{ $zone }}. 10800 IN A {{ . }}
*.{{ $zone }}. 10800 IN A {{ . }}
*.*.{{ $zone }}. 10800 IN A {{ . }}
{{ end }}
*.p.{{ $zone }}. 10800 IN A {{ .privateIngressIP }}
`)
	records, err := fs.Create("zone.db")
	if err != nil {
		return err
	}
	defer records.Close()
	if err := recordsTmpl.Execute(records, map[string]any{
		"zone":             zone.Zone,
		"publicIngressIPs": zone.PublicIPs,
		"privateIngressIP": zone.PrivateIP,
		"nameservers":      zone.Nameservers,
		"nowUnix":          NowUnix(),
	}); err != nil {
		return err
	}
	return nil
}

func (s *fsZoneStore) ConfigPath() string {
	return s.fs.Join(s.fs.Root(), zoneConfigFilename)
}

func (s *fsZoneStore) AddDNSSec(key DNSSecKey) error {
	return nil
}

func (s *fsZoneStore) AddTextRecord(entry, txt string) error {
	s.fs.Remove("txt")
	r, err := s.fs.Open("zone.db")
	if err != nil {
		return err
	}
	defer r.Close()
	z, err := NewZoneFile(r)
	if err != nil {
		return err
	}
	var fqdn = fmt.Sprintf("%s.%s.", entry, s.zone.Zone)
	z.CreateOrReplaceTxtRecord(fqdn, txt)
	for _, ip := range s.zone.PublicIPs {
		z.CreateARecord(fqdn, ip)
	}
	w, err := s.fs.Create("zone.db")
	if err != nil {
		return err
	}
	defer w.Close()
	if err := z.Write(w); err != nil {
		return err
	}
	return nil
}

func (s *fsZoneStore) DeleteTextRecord(entry, txt string) error {
	r, err := s.fs.Open("zone.db")
	if err != nil {
		return err
	}
	defer r.Close()
	z, err := NewZoneFile(r)
	if err != nil {
		return err
	}
	fqdn := fmt.Sprintf("%s.%s.", entry, s.zone.Zone)
	z.DeleteTxtRecord(fqdn, txt)
	z.DeleteRecordsFor(fqdn)
	w, err := s.fs.Create("zone.db")
	if err != nil {
		return err
	}
	defer w.Close()
	if err := z.Write(w); err != nil {
		return err
	}
	return nil
}
