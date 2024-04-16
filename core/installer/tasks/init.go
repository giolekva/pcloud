package tasks

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

func SetupConfigRepoTask(env Env, st *state) Task {
	ret := newSequentialParentTask(
		"Configure Git repository",
		true,
		newSequentialParentTask(
			"Start up Git server",
			false,
			NewCreateConfigRepoTask(env, st),
			CreateGitClientTask(env, st),
		),
		NewInitConfigRepoTask(env, st),
		NewActivateEnvTask(env, st),
		newSequentialParentTask(
			"Create initial commit",
			false,
			CreateRepoClient(env, st),
			CommitEnvironmentConfiguration(env, st),
			ConfigureFirstAccount(env, st),
		),
	)
	ret.beforeStart = func() {
		st.infoListener("dodo is driven by GitOps, changes are committed to the repository before updating an environment. This unlocks functionalities such as: rolling back to old working state, migrating dodo to new infrastructure (for example from Cloud to on-prem).")
	}
	return ret
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
		derived := installer.Derived{
			Global: installer.Values{
				Id:            env.Name,
				PCloudEnvName: env.PCloudEnvName,
			},
			Values: map[string]any{
				"privateKey": string(ssKeys.RawPrivateKey()),
				"publicKey":  string(ssKeys.RawAuthorizedKey()),
				"adminKey":   string(ssAdminKeys.RawAuthorizedKey()),
			},
		}
		return installer.InstallApp(st.repo, st.nsCreator, ssApp, filepath.Join("/environments", env.Name, "config-repo"), env.Name, derived.Values, derived)
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
	t := newLeafTask("Configure access control lists", func() error {
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
		repoIO, err := installer.NewRepoIO(repo, st.ssClient.Signer)
		if err != nil {
			return err
		}
		if err := repoIO.Atomic(func(r installer.RepoFS) (string, error) {
			w, err := r.Writer("README.md")
			if err != nil {
				return "", err
			}
			defer w.Close()
			if _, err := fmt.Fprintf(w, "# %s PCloud environment", env.Name); err != nil {
				return "", err
			}
			if err := installer.WriteYaml(r, "kustomization.yaml", installer.NewKustomization()); err != nil {
				return "", err
			}
			return "init", nil
		}); err != nil {
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
