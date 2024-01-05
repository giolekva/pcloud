package tasks

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

func SetupConfigRepoTask(env Env, st *state) Task {
	return newSequentialParentTask(
		"Configuration repository",
		NewCreateConfigRepoTask(env, st),
		CreateGitClientTask(env, st),
		NewInitConfigRepoTask(env, st),
	)
}

func NewCreateConfigRepoTask(env Env, st *state) Task {
	t := newLeafTask("Install Git server", func() error {
		appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
		ssApp, err := appsRepo.Find("config-repo")
		if err != nil {
			return err
		}
		ssAdminKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-admin-keys", env.Name))
		if err != nil {
			return err
		}
		st.ssAdminKeys = ssAdminKeys
		ssKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-keys", env.Name))
		if err != nil {
			return err
		}
		ssValues := map[string]any{
			"privateKey": string(ssKeys.RawPrivateKey()),
			"publicKey":  string(ssKeys.RawAuthorizedKey()),
			"adminKey":   string(ssAdminKeys.RawAuthorizedKey()),
		}
		derived := installer.Derived{
			Global: installer.Values{
				Id:            env.Name,
				PCloudEnvName: env.PCloudEnvName,
			},
			Release: installer.Release{
				Namespace: env.Name,
			},
			Values: ssValues,
		}
		if err := st.nsCreator.Create(env.Name); err != nil {
			return err
		}
		if err := st.repo.InstallApp(*ssApp, filepath.Join("/environments", env.Name, "config-repo"), ssValues, derived); err != nil {
			return err
		}
		return nil
	})
	return &t
}

func CreateGitClientTask(env Env, st *state) Task {
	t := newLeafTask("Wait git server to come up", func() error {
		ssClient, err := soft.WaitForClient(
			fmt.Sprintf("soft-serve.%s.svc.cluster.local:%d", env.Name, 22),
			st.ssAdminKeys.RawPrivateKey(),
			log.Default())
		if err != nil {
			return err
		}
		if err := ssClient.AddPublicKey("admin", env.AdminPublicKey); err != nil {
			return err
		}
		// // TODO(gio): defer?
		// // TODO(gio): remove at the end of final task cleanup
		// if err := ssClient.RemovePublicKey("admin", string(ssAdminKeys.RawAuthorizedKey())); err != nil {
		// 	t.callDoneListeners(err)
		// 	return
		// }
		st.ssClient = ssClient
		return nil
	})
	return &t
}

func NewInitConfigRepoTask(env Env, st *state) Task {
	t := newLeafTask("Create Git repository for environment configuration", func() error {
		st.fluxUserName = fmt.Sprintf("flux-%s", env.Name)
		keys, err := installer.NewSSHKeyPair(st.fluxUserName)
		if err != nil {
			return err
		}
		st.keys = keys
		if err := st.ssClient.AddRepository("config"); err != nil {
			return err
		}
		repo, err := st.ssClient.GetRepo("config")
		if err != nil {
			return err
		}
		repoIO := installer.NewRepoIO(repo, st.ssClient.Signer)
		if err := repoIO.WriteCommitAndPush("README.md", fmt.Sprintf("# %s PCloud environment", env.Name), "readme"); err != nil {
			return err
		}
		if err := st.ssClient.AddUser(st.fluxUserName, keys.AuthorizedKey()); err != nil {
			return err
		}
		if err := st.ssClient.AddReadOnlyCollaborator("config", st.fluxUserName); err != nil {
			return err
		}
		return nil
	})
	return &t
}
