package tasks

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

type createConfigRepoTask struct {
	basicTask
	env Env
	st  *state
}

func NewCreateConfigRepoTask(env Env, st *state) Task {
	return &createConfigRepoTask{
		basicTask: basicTask{
			title: "Install Git server",
		},
		env: env,
		st:  st,
	}
}

func (t *createConfigRepoTask) Start() {
	appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
	ssApp, err := appsRepo.Find("soft-serve")
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	ssAdminKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-admin-keys", t.env.Name))
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	ssKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-keys", t.env.Name))
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	ssValues := map[string]any{
		"ChartRepositoryNamespace": t.env.PCloudEnvName,
		"ServiceType":              "ClusterIP",
		"PrivateKey":               string(ssKeys.RawPrivateKey()),
		"PublicKey":                string(ssKeys.RawAuthorizedKey()),
		"AdminKey":                 string(ssAdminKeys.RawAuthorizedKey()),
		"Ingress": map[string]any{
			"Enabled": false,
		},
	}
	derived := installer.Derived{
		Global: installer.Values{
			Id:            t.env.Name,
			PCloudEnvName: t.env.PCloudEnvName,
		},
		Release: installer.Release{
			Namespace: t.env.Name,
		},
		Values: ssValues,
	}
	if err := t.st.nsCreator.Create(t.env.Name); err != nil {
		t.callDoneListeners(err)
		return
	}
	if err := t.st.repo.InstallApp(*ssApp, filepath.Join("/environments", t.env.Name, "config-repo"), ssValues, derived); err != nil {
		t.callDoneListeners(err)
		return
	}
	ssClient, err := soft.WaitForClient(
		fmt.Sprintf("soft-serve.%s.svc.cluster.local:%d", t.env.Name, 22),
		ssAdminKeys.RawPrivateKey(),
		log.Default())
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	if err := ssClient.AddPublicKey("admin", t.env.AdminPublicKey); err != nil {
		t.callDoneListeners(err)
		return
	}
	// // TODO(gio): defer?
	// // TODO(gio): remove at the end of final task cleanup
	// if err := ssClient.RemovePublicKey("admin", string(ssAdminKeys.RawAuthorizedKey())); err != nil {
	// 	t.callDoneListeners(err)
	// 	return
	// }
	t.st.ssClient = ssClient
	t.callDoneListeners(nil)
}

type initConfigRepoTask struct {
	basicTask
	env Env
	st  *state
}

func NewInitConfigRepoTask(env Env, st *state) Task {
	return &initConfigRepoTask{
		basicTask: basicTask{
			title: "Create Git repository for environment configuration",
		},
		env: env,
		st:  st,
	}
}

func (t *initConfigRepoTask) Start() {
	t.st.fluxUserName = fmt.Sprintf("flux-%s", t.env.Name)
	keys, err := installer.NewSSHKeyPair(t.st.fluxUserName)
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	t.st.keys = keys
	if err := t.st.ssClient.AddRepository("config"); err != nil {
		t.callDoneListeners(err)
		return
	}
	repo, err := t.st.ssClient.GetRepo("config")
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	repoIO := installer.NewRepoIO(repo, t.st.ssClient.Signer)
	if err := repoIO.WriteCommitAndPush("README.md", fmt.Sprintf("# %s PCloud environment", t.env.Name), "readme"); err != nil {
		t.callDoneListeners(err)
		return
	}
	if err := t.st.ssClient.AddUser(t.st.fluxUserName, keys.AuthorizedKey()); err != nil {
		t.callDoneListeners(err)
		return
	}
	if err := t.st.ssClient.AddReadOnlyCollaborator("config", t.st.fluxUserName); err != nil {
		t.callDoneListeners(err)
		return
	}
	t.callDoneListeners(nil)
}
