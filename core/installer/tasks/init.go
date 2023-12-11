package tasks

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

func NewCreateConfigRepoTask(env Env, st *state) Task {
	t := newLeafTask("Install Git server", func() error {
		appsRepo := installer.NewInMemoryAppRepository(installer.CreateAllApps())
		ssApp, err := appsRepo.Find("soft-serve")
		if err != nil {
			return err
		}
		ssAdminKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-admin-keys", env.Name))
		if err != nil {
			return err
		}
		ssKeys, err := installer.NewSSHKeyPair(fmt.Sprintf("%s-config-repo-keys", env.Name))
		if err != nil {
			return err
		}
		ssValues := map[string]any{
			"ChartRepositoryNamespace": env.PCloudEnvName,
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
		ssClient, err := soft.WaitForClient(
			fmt.Sprintf("soft-serve.%s.svc.cluster.local:%d", env.Name, 22),
			ssAdminKeys.RawPrivateKey(),
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
