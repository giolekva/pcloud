package tasks

import (
	"net"

	"github.com/charmbracelet/keygen"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

type state struct {
	publicIPs    []net.IP
	nsCreator    installer.NamespaceCreator
	repo         installer.RepoIO
	ssClient     *soft.Client
	fluxUserName string
	keys         *keygen.KeyPair
}

type Env struct {
	PCloudEnvName  string
	Name           string
	ContactEmail   string
	Domain         string
	AdminPublicKey string
}

func NewCreateEnvTask(
	env Env,
	publicIPs []net.IP,
	nsCreator installer.NamespaceCreator,
	repo installer.RepoIO,
) Task {
	st := state{
		publicIPs: publicIPs,
		nsCreator: nsCreator,
		repo:      repo,
	}
	t := newSequentialParentTask(
		"Create env",
		NewCreateConfigRepoTask(env, &st),
		NewInitConfigRepoTask(env, &st),
		NewActivateEnvTask(env, &st),
		NewDNSResolverTask(env.Domain, publicIPs, env, &st),
		NewSetupInfraAppsTask(env, &st),
	)
	return &t
}
