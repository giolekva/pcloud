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
	infoListener    EnvInfoListener
	publicIPs       []net.IP
	nsCreator       installer.NamespaceCreator
	repo            installer.RepoIO
	ssAdminKeys     *keygen.KeyPair
	ssClient        *soft.Client
	fluxUserName    string
	keys            *keygen.KeyPair
	appManager      *installer.AppManager
	appsRepo        installer.AppRepository
	infraAppManager *installer.InfraAppManager
}

type Env struct {
	PCloudEnvName   string
	Name            string
	ContactEmail    string
	Domain          string
	AdminPublicKey  string
	NamespacePrefix string
}

type EnvInfoListener func(string)

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
	mgr *installer.InfraAppManager,
	infoListener EnvInfoListener,
) (Task, DNSZoneRef) {
	st := state{
		infoListener:    infoListener,
		publicIPs:       publicIPs,
		nsCreator:       nsCreator,
		repo:            repo,
		infraAppManager: mgr,
	}
	t := newSequentialParentTask(
		"Create env",
		true,
		SetupConfigRepoTask(env, &st),
		SetupZoneTask(env, startIP, &st),
		SetupInfra(env, startIP, &st),
	)
	t.afterDone = func() {
		infoListener(fmt.Sprintf("dodo environment for %s has been provisioned successfully. Visit [https://welcome.%s](https://welcome.%s) to create administrative account and log into the system.", env.Domain, env.Domain, env.Domain))
	}
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
