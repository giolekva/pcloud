package tasks

import (
	"context"
	"fmt"

	"github.com/charmbracelet/keygen"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/dns"
	"github.com/giolekva/pcloud/core/installer/http"
	"github.com/giolekva/pcloud/core/installer/soft"
)

type state struct {
	infoListener    EnvInfoListener
	nsCreator       installer.NamespaceCreator
	jc              installer.JobCreator
	hf              installer.HelmFetcher
	dnsFetcher      installer.ZoneStatusFetcher
	httpClient      http.Client
	dnsClient       dns.Client
	repo            soft.RepoIO
	repoClient      soft.ClientGetter
	ssAdminKeys     *keygen.KeyPair
	ssClient        soft.Client
	fluxUserName    string
	keys            *keygen.KeyPair
	appManager      *installer.AppManager
	appsRepo        installer.AppRepository
	infraAppManager *installer.InfraAppManager
}

type EnvInfoListener func(string)

func NewCreateEnvTask(
	env installer.EnvConfig,
	nsCreator installer.NamespaceCreator,
	jc installer.JobCreator,
	hf installer.HelmFetcher,
	dnsFetcher installer.ZoneStatusFetcher,
	httpClient http.Client,
	dnsClient dns.Client,
	repo soft.RepoIO,
	repoClient soft.ClientGetter,
	mgr *installer.InfraAppManager,
	infoListener EnvInfoListener,
) (Task, installer.EnvDNS) {
	st := state{
		infoListener:    infoListener,
		nsCreator:       nsCreator,
		jc:              jc,
		hf:              hf,
		dnsFetcher:      dnsFetcher,
		httpClient:      httpClient,
		dnsClient:       dnsClient,
		repo:            repo,
		repoClient:      repoClient,
		infraAppManager: mgr,
	}
	t := newSequentialParentTask(
		"Create env",
		true,
		SetupConfigRepoTask(env, &st),
		SetupZoneTask(env, mgr, &st),
		SetupInfra(env, &st),
	)
	t.afterDone = func() {
		infoListener(fmt.Sprintf("dodo environment for %s has been provisioned successfully. Visit [https://welcome.%s](https://welcome.%s) to create administrative account and log into the system.", env.Domain, env.Domain, env.Domain))
	}
	rctx, done := context.WithCancel(context.Background())
	t.OnDone(func(_ error) {
		done()
	})
	pr := NewFixedReconciler(
		fmt.Sprintf("%s-flux", env.InfraName),
		fmt.Sprintf("%s-flux", env.InfraName),
	)
	er := NewFixedReconciler(
		env.Id,
		env.Id,
	)
	go pr.Reconcile(rctx)
	go er.Reconcile(rctx)
	return t, installer.EnvDNS{
		Zone:    env.Domain,
		Address: fmt.Sprintf("http://dns-api.%sdns.svc.cluster.local/records-to-publish", env.NamespacePrefix),
	}
}
