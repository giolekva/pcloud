package welcome

import (
	"bytes"
	"encoding/json"
	"golang.org/x/crypto/ssh"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	// "github.com/go-git/go-git/v5"
	// "github.com/go-git/go-git/v5/storage/memory"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/tasks"
)

type fakeNSCreator struct {
	t *testing.T
}

func (f fakeNSCreator) Create(name string) error {
	f.t.Logf("Create namespace: %s", name)
	return nil
}

type fakeJobCreator struct {
	t *testing.T
}

func (f fakeJobCreator) Create(name, namespace string, image string, cmd []string) error {
	f.t.Logf("Create job: %s/%s %s \"%s\"", namespace, name, image, strings.Join(cmd, " "))
	return nil
}

type fakeHelmFetcher struct {
	t *testing.T
}

func (f fakeHelmFetcher) Pull(chart installer.HelmChartGitRepo, rfs soft.RepoFS, root string) error {
	f.t.Logf("Helm pull: %+v", chart)
	return nil
}

type fakeZoneStatusFetcher struct {
	t *testing.T
}

func (f fakeZoneStatusFetcher) Fetch(addr string) (string, error) {
	f.t.Logf("Fetching status: %s", addr)
	return addr, nil
}

type mockRepoIO struct {
	soft.RepoFS
	addr string
	t    *testing.T
	l    sync.Locker
}

func (r mockRepoIO) FullAddress() string {
	return r.addr
}

func (r mockRepoIO) Pull() error {
	r.t.Logf("Pull: %s", r.addr)
	return nil
}

func (r mockRepoIO) CommitAndPush(message string, opts ...soft.PushOption) error {
	r.t.Logf("Commit and push: %s", message)
	return nil
}

func (r mockRepoIO) Do(op soft.DoFn, _ ...soft.DoOption) error {
	r.l.Lock()
	defer r.l.Unlock()
	msg, err := op(r)
	if err != nil {
		return err
	}
	return r.CommitAndPush(msg)
}

type fakeSoftServeClient struct {
	t     *testing.T
	envFS billy.Filesystem
}

func (f fakeSoftServeClient) Address() string {
	return ""
}

func (f fakeSoftServeClient) Signer() ssh.Signer {
	return nil
}

func (f fakeSoftServeClient) GetPublicKeys() ([]string, error) {
	return []string{}, nil
}

func (f fakeSoftServeClient) GetRepo(name string) (soft.RepoIO, error) {
	var l sync.Mutex
	return mockRepoIO{soft.NewBillyRepoFS(f.envFS), "foo.bar", f.t, &l}, nil
}

func (f fakeSoftServeClient) GetRepoAddress(name string) string {
	return ""
}

func (f fakeSoftServeClient) AddRepository(name string) error {
	return nil
}

func (f fakeSoftServeClient) AddUser(name, pubKey string) error {
	return nil
}

func (f fakeSoftServeClient) AddPublicKey(user string, pubKey string) error {
	return nil
}

func (f fakeSoftServeClient) RemovePublicKey(user string, pubKey string) error {
	return nil
}

func (f fakeSoftServeClient) MakeUserAdmin(name string) error {
	return nil
}

func (f fakeSoftServeClient) AddReadWriteCollaborator(repo, user string) error {
	return nil
}

func (f fakeSoftServeClient) AddReadOnlyCollaborator(repo, user string) error {
	return nil
}

func (f fakeSoftServeClient) AddWebhook(repo, url string, opts ...string) error {
	return nil
}

type fakeClientGetter struct {
	t     *testing.T
	envFS billy.Filesystem
}

func (f fakeClientGetter) Get(addr string, clientPrivateKey []byte, log *log.Logger) (soft.Client, error) {
	return fakeSoftServeClient{f.t, f.envFS}, nil
}

const infraConfig = `
infraAdminPublicKey: Zm9vYmFyCg==
namespacePrefix: infra-
pcloudEnvName: infra
publicIP:
- 1.1.1.1
- 2.2.2.2
`

const envCidrs = ``

type fixedNameGenerator struct{}

func (f fixedNameGenerator) Generate() (string, error) {
	return "test", nil
}

type fakeHttpClient struct {
	t      *testing.T
	counts map[string]int
}

func (f fakeHttpClient) Get(addr string) (*http.Response, error) {
	f.t.Logf("HTTP GET: %s", addr)
	cnt, ok := f.counts[addr]
	if !ok {
		cnt = 0
	}
	f.counts[addr] = cnt + 1
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Body:       io.NopCloser(strings.NewReader("ok")),
	}, nil
}

type fakeDnsClient struct {
	t      *testing.T
	counts map[string]int
}

func (f fakeDnsClient) Lookup(host string) ([]net.IP, error) {
	f.t.Logf("HTTP GET: %s", host)
	return []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2")}, nil
}

type onDoneTaskMap struct {
	m      tasks.TaskManager
	onDone tasks.TaskDoneListener
}

func (m *onDoneTaskMap) Add(name string, task tasks.Task) error {
	if err := m.m.Add(name, task); err != nil {
		return err
	} else {
		task.OnDone(m.onDone)
		return nil
	}
}

func (m *onDoneTaskMap) Get(name string) (tasks.Task, error) {
	return m.m.Get(name)
}

func TestCreateNewEnv(t *testing.T) {
	apps := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	infraFS := memfs.New()
	envFS := memfs.New()
	nsCreator := fakeNSCreator{t}
	jc := fakeJobCreator{t}
	hf := fakeHelmFetcher{t}
	lg := installer.GitRepositoryLocalChartGenerator{"foo", "bar"}
	infraRepo := mockRepoIO{soft.NewBillyRepoFS(infraFS), "foo.bar", t, &sync.Mutex{}}
	infraMgr, err := installer.NewInfraAppManager(infraRepo, nsCreator, hf, lg)
	if err != nil {
		t.Fatal(err)
	}
	if err := util.WriteFile(infraFS, "config.yaml", []byte(infraConfig), fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := util.WriteFile(infraFS, "env-cidrs.yaml", []byte(envCidrs), fs.ModePerm); err != nil {
		t.Fatal(err)
	}
	{
		app, err := installer.FindInfraApp(apps, "dns-gateway")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := infraMgr.Install(app, "/infrastructure/dns-gateway", "dns-gateway", map[string]any{
			"servers": []installer.EnvDNS{},
		}); err != nil {
			t.Fatal(err)
		}
	}
	cg := fakeClientGetter{t, envFS}
	httpClient := fakeHttpClient{t, make(map[string]int)}
	dnsClient := fakeDnsClient{t, make(map[string]int)}
	var done sync.WaitGroup
	done.Add(1)
	var taskErr error
	tm := &onDoneTaskMap{
		tasks.NewTaskMap(),
		func(err error) {
			taskErr = err
			done.Done()
		},
	}
	s := NewEnvServer(
		8181,
		fakeSoftServeClient{t, envFS},
		infraRepo,
		cg,
		nsCreator,
		jc,
		hf,
		fakeZoneStatusFetcher{t},
		fixedNameGenerator{},
		httpClient,
		dnsClient,
		tm,
	)
	go s.Start()
	time.Sleep(1 * time.Second) // Let server start
	req := createEnvReq{
		Name:           "test",
		ContactEmail:   "test@test.t",
		Domain:         "test.t",
		AdminPublicKey: "test",
		SecretToken:    "test",
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post("http://localhost:8181/", "application/json", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		io.Copy(&buf, resp.Body)
		t.Fatal(buf.String())
	}
	done.Wait()
	http.Get("http://localhost:8181/env/test")
	debugFS(infraFS, t, "/infrastructure/dns-gateway/resources/coredns.yaml")
	debugFS(envFS, t)
	if taskErr != nil {
		t.Fatal(taskErr)
	}
	expected := []string{
		"https://accounts-ui.test.t",
		"https://welcome.test.t",
		"https://memberships.p.test.t",
		"https://launcher.test.t",
		"https://headscale.test.t/apple",
	}
	for _, e := range expected {
		if cnt, ok := httpClient.counts[e]; !ok || cnt != 1 {
			t.Fatal(httpClient.counts)
		}
	}
	if len(httpClient.counts) != 5 {
		t.Fatal(httpClient.counts)
	}
}

func debugFS(bfs billy.Filesystem, t *testing.T, files ...string) {
	f := map[string]struct{}{}
	for _, i := range files {
		f[i] = struct{}{}
	}
	t.Log("----- START ------")
	err := util.Walk(bfs, "/", func(path string, info fs.FileInfo, err error) error {
		if _, ok := f[path]; ok && !info.IsDir() {
			contents, err := util.ReadFile(bfs, path)
			if err != nil {
				return err
			}
			t.Log(string(contents))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("----- END ------")
}
