package tasks

import (
	"context"
	"fmt"
	"net"

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
	appsRepo       installer.AppRepository
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
	startIP net.IP,
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
			SetupInfra(env, startIP, &st)...,
		)...,
	)
	rctx, done := context.WithCancel(context.Background())
	t.OnDone(func(_ error) {
		done()
	})
	pr := NewFluxcdReconciler( // TODO(gio): make reconciler address a flag
		"http://fluxcd-reconciler.dodo-fluxcd-reconciler.svc.cluster.local",
		fmt.Sprintf("%s-flux", env.PCloudEnvName),
	)
	er := NewFluxcdReconciler(
		"http://fluxcd-reconciler.dodo-fluxcd-reconciler.svc.cluster.local",
		env.Name,
	)
	go pr.Reconcile(rctx)
	go er.Reconcile(rctx)
	return t, DNSZoneRef{"dns-zone", env.Name}
}
