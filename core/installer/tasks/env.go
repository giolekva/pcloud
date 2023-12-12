package tasks

import (
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
	return newSequentialParentTask(
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
}
