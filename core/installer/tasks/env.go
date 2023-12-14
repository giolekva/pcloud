package tasks

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/charmbracelet/keygen"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

type state struct {
	publicIPs      []net.IP
	nsCreator      installer.NamespaceCreator
	repo           installer.RepoIO
	ssAdminKeys    *keygen.KeyPair
	ssClient       *soft.Client
	fluxUserName   string
	keys           *keygen.KeyPair
	appManager     *installer.AppManager
	appsRepo       installer.AppRepository[installer.App]
	nsGen          installer.NamespaceGenerator
	emptySuffixGen installer.SuffixGenerator
}

type Env struct {
	PCloudEnvName  string
	Name           string
	ContactEmail   string
	Domain         string
	AdminPublicKey string
}

type DNSZoneRef struct {
	Name      string
	Namespace string
}

func NewCreateEnvTask(
	env Env,
	publicIPs []net.IP,
	nsCreator installer.NamespaceCreator,
	repo installer.RepoIO,
) (Task, DNSZoneRef) {
	st := state{
		publicIPs: publicIPs,
		nsCreator: nsCreator,
		repo:      repo,
	}
	t := newSequentialParentTask(
		"Create env",
		append(
			[]Task{
				SetupConfigRepoTask(env, &st),
				NewActivateEnvTask(env, &st),
				SetupZoneTask(env, &st),
			},
			SetupInfra(env, &st)...,
		)...,
	)
	done := make(chan struct{})
	t.OnDone(func(_ error) {
		close(done)
	})
	go reconcile(fmt.Sprintf("%s-flux", env.PCloudEnvName), done)
	go reconcile(env.Name, done)
	return t, DNSZoneRef{"dns-zone", env.Name}
}

func reconcile(name string, quit chan struct{}) {
	git := fmt.Sprintf("http://fluxcd-reconciler.dodo-fluxcd-reconciler.svc.cluster.local/source/git/%s/%s/reconcile", name, name)
	kust := fmt.Sprintf("http://fluxcd-reconciler.dodo-fluxcd-reconciler.svc.cluster.local/kustomization/%s/%s/reconcile", name, name)
	for {
		select {
		case <-time.After(30 * time.Second):
			http.Get(git)
			http.Get(kust)
		case <-quit:
			return
		}
	}
}
