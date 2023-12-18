package welcome

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	htemplate "html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/tasks"
)

//go:embed create-env.html
var createEnvFormHtml []byte

//go:embed env-created.html
var envCreatedHtml string

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusAccepted Status = "ACCEPTED"
)

// TODO(giolekva): add CreatedAt and ValidUntil
type invitation struct {
	Token  string `json:"token"`
	Status Status `json:"status"`
}

type EnvServer struct {
	port          int
	ss            *soft.Client
	repo          installer.RepoIO
	nsCreator     installer.NamespaceCreator
	dnsFetcher    installer.ZoneStatusFetcher
	nameGenerator installer.NameGenerator
	tasks         map[string]tasks.Task
	dns           map[string]tasks.DNSZoneRef
}

func NewEnvServer(
	port int,
	ss *soft.Client,
	repo installer.RepoIO,
	nsCreator installer.NamespaceCreator,
	dnsFetcher installer.ZoneStatusFetcher,
	nameGenerator installer.NameGenerator,
) *EnvServer {
	return &EnvServer{
		port,
		ss,
		repo,
		nsCreator,
		dnsFetcher,
		nameGenerator,
		make(map[string]tasks.Task),
		make(map[string]tasks.DNSZoneRef),
	}
}

func (s *EnvServer) Start() {
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticAssets)))
	r.Path("/env/{key}").Methods("GET").HandlerFunc(s.monitorTask)
	r.Path("/env/{key}").Methods("POST").HandlerFunc(s.publishDNSRecords)
	r.Path("/").Methods("GET").HandlerFunc(s.createEnvForm)
	r.Path("/").Methods("POST").HandlerFunc(s.createEnv)
	r.Path("/create-invitation").Methods("GET").HandlerFunc(s.createInvitation)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func (s *EnvServer) monitorTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		http.Error(w, "Task key not provided", http.StatusBadRequest)
		return
	}
	t, ok := s.tasks[key]
	if !ok {
		http.Error(w, "Task not found", http.StatusBadRequest)
		return
	}
	dnsRef, ok := s.dns[key]
	if !ok {
		http.Error(w, "Task dns configuration not found", http.StatusInternalServerError)
		return
	}
	err, ready, info := s.dnsFetcher.Fetch(dnsRef.Namespace, dnsRef.Name)
	// TODO(gio): check error type
	if err != nil && (ready || len(info.Records) > 0) {
		panic("!! SHOULD NOT REACH !!")
	}
	if !ready && len(info.Records) > 0 {
		panic("!! SHOULD NOT REACH !!")
	}
	tmpl, err := htemplate.New("response").Parse(envCreatedHtml)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, map[string]any{
		"Root":       t,
		"DNSRecords": info.Records,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *EnvServer) publishDNSRecords(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		http.Error(w, "Task key not provided", http.StatusBadRequest)
		return
	}
	dnsRef, ok := s.dns[key]
	if !ok {
		http.Error(w, "Task dns configuration not found", http.StatusInternalServerError)
		return
	}
	err, ready, info := s.dnsFetcher.Fetch(dnsRef.Namespace, dnsRef.Name)
	// TODO(gio): check error type
	if err != nil && (ready || len(info.Records) > 0) {
		panic("!! SHOULD NOT REACH !!")
	}
	if !ready && len(info.Records) > 0 {
		panic("!! SHOULD NOT REACH !!")
	}
	r.ParseForm()
	if apiToken, err := getFormValue(r.PostForm, "api-token"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		p := NewGandiUpdater(apiToken)
		zone := strings.Join(strings.Split(info.Zone, ".")[1:], ".") // TODO(gio): this is not gonna work with no subdomain case
		if err := p.Update(zone, strings.Split(info.Records, "\n")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, fmt.Sprintf("/env/%s", key), http.StatusSeeOther)
}

func (s *EnvServer) createEnvForm(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write(createEnvFormHtml); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *EnvServer) createInvitation(w http.ResponseWriter, r *http.Request) {
	invitations, err := s.readInvitations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	token, err := installer.NewFixedLengthRandomNameGenerator(100).Generate() // TODO(giolekva): use cryptographic tokens
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return

	}
	invitations = append(invitations, invitation{token, StatusActive})
	if err := s.writeInvitations(invitations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type createEnvReq struct {
	Name           string
	ContactEmail   string `json:"contactEmail"`
	Domain         string `json:"domain"`
	AdminPublicKey string `json:"adminPublicKey"`
	SecretToken    string `json:"secretToken"`
}

func (s *EnvServer) readInvitations() ([]invitation, error) {
	r, err := s.repo.Reader("invitations")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return make([]invitation, 0), nil
		}
		return nil, err
	}
	defer r.Close()
	dec := json.NewDecoder(r)
	invitations := make([]invitation, 0)
	for {
		var i invitation
		if err := dec.Decode(&i); err == io.EOF {
			break
		}
		invitations = append(invitations, i)
	}
	return invitations, nil
}

func (s *EnvServer) writeInvitations(invitations []invitation) error {
	w, err := s.repo.Writer("invitations")
	if err != nil {
		return err
	}
	defer w.Close()
	enc := json.NewEncoder(w)
	for _, i := range invitations {
		if err := enc.Encode(i); err != nil {
			return err
		}
	}
	return s.repo.CommitAndPush("Generated new invitation")
}

func extractRequest(r *http.Request) (createEnvReq, error) {
	var req createEnvReq
	if err := func() error {
		var err error
		if err = r.ParseForm(); err != nil {
			return err
		}
		if req.SecretToken, err = getFormValue(r.PostForm, "secret-token"); err != nil {
			return err
		}
		if req.Domain, err = getFormValue(r.PostForm, "domain"); err != nil {
			return err
		}
		if req.ContactEmail, err = getFormValue(r.PostForm, "contact-email"); err != nil {
			return err
		}
		if req.AdminPublicKey, err = getFormValue(r.PostForm, "admin-public-key"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return createEnvReq{}, err
		}
	}
	return req, nil
}

func (s *EnvServer) acceptInvitation(token string) error {
	invitations, err := s.readInvitations()
	if err != nil {
		return err
	}
	found := false
	for i := range invitations {
		if invitations[i].Token == token && invitations[i].Status == StatusActive {
			invitations[i].Status = StatusAccepted
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Invitation not found")
	}
	return s.writeInvitations(invitations)
}

func (s *EnvServer) createEnv(w http.ResponseWriter, r *http.Request) {
	req, err := extractRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var env installer.EnvConfig
	cr, err := s.repo.Reader("config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cr.Close()
	if err := installer.ReadYaml(cr, &env); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// if err := s.acceptInvitation(req.SecretToken); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	if name, err := s.nameGenerator.Generate(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		req.Name = name
	}
	t, dns := tasks.NewCreateEnvTask(
		tasks.Env{
			PCloudEnvName:  env.Name,
			Name:           req.Name,
			ContactEmail:   req.ContactEmail,
			Domain:         req.Domain,
			AdminPublicKey: req.AdminPublicKey,
		},
		[]net.IP{
			net.ParseIP("135.181.48.180"),
			net.ParseIP("65.108.39.172"),
		},
		s.nsCreator,
		s.repo,
	)
	key := func() string {
		for {
			key, err := s.nameGenerator.Generate()
			if err == nil {
				return key
			}
		}
	}()
	s.tasks[key] = t
	s.dns[key] = dns
	go t.Start()
	http.Redirect(w, r, fmt.Sprintf("/env/key", key), http.StatusSeeOther)
}
