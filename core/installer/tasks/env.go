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
	publicIPs    []net.IP
	nsCreator    installer.NamespaceCreator
	repo         installer.RepoIO
	ssClient     *soft.Client
	fluxUserName string
	keys         *keygen.KeyPair
}

type createEnvTask struct {
	basicTask
	env              Env
	st               state
	createConfigRepo Task
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
	ctx := context.Background()
	e := &createEnvTask{
		basicTask: basicTask{
			title: fmt.Sprintf("Create %s environment", env.Domain),
		},
		env: env,
		st: state{
			publicIPs: publicIPs,
			nsCreator: nsCreator,
			repo:      repo,
		},
	}
	e.createConfigRepo = NewCreateConfigRepoTask(env, &e.st)
	e.AddSubtask(e.createConfigRepo)
	initRepo := NewInitConfigRepoTask(env, &e.st)
	e.AddSubtask(initRepo)
	e.createConfigRepo.OnDone(func(err error) {
		if err == nil {
			initRepo.Start()
		} else {
			e.callDoneListeners(err)
		}
	})
	activate := NewActivateEnvTask(env, &e.st)
	e.AddSubtask(activate)
	initRepo.OnDone(func(err error) {
		if err == nil {
			activate.Start()
		} else {
			e.callDoneListeners(err)
		}
	})
	dns := NewDNSResolverTask(env.Domain, publicIPs, ctx, env, &e.st)
	e.AddSubtask(dns)
	activate.OnDone(func(err error) {
		if err == nil {
			dns.Start()
		} else {
			e.callDoneListeners(err)
		}
	})
	setupInfra := NewSetupInfraAppsTask(env, &e.st)
	e.AddSubtask(setupInfra)
	dns.OnDone(func(err error) {
		if err == nil {
			setupInfra.Start()
		} else {
			e.callDoneListeners(err)
		}
	})
	setupInfra.OnDone(func(err error) {
		e.callDoneListeners(err)
	})
	return e
}

func (t *createEnvTask) Start() {
	go t.createConfigRepo.Start()
}
