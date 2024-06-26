package welcome

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

type DodoAppServer struct {
	port      int
	sshKey    string
	client    soft.Client
	namespace string
	env       installer.EnvConfig
	jc        installer.JobCreator
	workers   map[string]struct{}
}

func NewDodoAppServer(
	port int,
	sshKey string,
	client soft.Client,
	namespace string,
	jc installer.JobCreator,
	env installer.EnvConfig,
) *DodoAppServer {
	return &DodoAppServer{
		port,
		sshKey,
		client,
		namespace,
		env,
		jc,
		map[string]struct{}{},
	}
}

func (s *DodoAppServer) Start() error {
	http.HandleFunc("/update", s.handleUpdate)
	http.HandleFunc("/register-worker", s.handleRegisterWorker)
	http.HandleFunc("/api/add-admin-key", s.handleAddAdminKey)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

type updateReq struct {
	Ref string `json:"ref"`
}

func (s *DodoAppServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("update")
	var req updateReq
	var contents strings.Builder
	io.Copy(&contents, r.Body)
	c := contents.String()
	fmt.Println(c)
	if err := json.NewDecoder(strings.NewReader(c)).Decode(&req); err != nil {
		fmt.Println(err)
		return
	}
	if req.Ref != "refs/heads/master" {
		return
	}
	go func() {
		time.Sleep(20 * time.Second)
		if err := UpdateDodoApp(s.client, s.namespace, s.sshKey, s.jc, &s.env); err != nil {
			fmt.Println(err)
		}
	}()
	for addr, _ := range s.workers {
		go func() {
			// TODO(gio): make port configurable
			http.Get(fmt.Sprintf("http://%s:3000/update", addr))
		}()
	}
}

type registerWorkerReq struct {
	Address string `json:"address"`
}

func (s *DodoAppServer) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	var req registerWorkerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.workers[req.Address] = struct{}{}
	fmt.Printf("registered worker: %s\n", req.Address)
}

type addAdminKeyReq struct {
	Public string `json:"public"`
}

func (s *DodoAppServer) handleAddAdminKey(w http.ResponseWriter, r *http.Request) {
	var req addAdminKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.client.AddPublicKey("admin", req.Public); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func UpdateDodoApp(client soft.Client, namespace string, sshKey string, jc installer.JobCreator, env *installer.EnvConfig) error {
	repo, err := client.GetRepo("app")
	if err != nil {
		return err
	}
	nsc := installer.NewNoOpNamespaceCreator()
	if err != nil {
		return err
	}
	hf := installer.NewGitHelmFetcher()
	m, err := installer.NewAppManager(repo, nsc, jc, hf, "/.dodo")
	if err != nil {
		return err
	}
	appCfg, err := soft.ReadFile(repo, "app.cue")
	fmt.Println(string(appCfg))
	if err != nil {
		return err
	}
	app, err := installer.NewDodoApp(appCfg)
	if err != nil {
		return err
	}
	lg := installer.GitRepositoryLocalChartGenerator{"app", namespace}
	if _, err := m.Install(app, "app", "/.dodo/app", namespace, map[string]any{
		"repoAddr":      repo.FullAddress(),
		"sshPrivateKey": sshKey,
	}, installer.WithConfig(env), installer.WithBranch("dodo"), installer.WithLocalChartGenerator(lg)); err != nil {
		return err
	}
	return nil
}
