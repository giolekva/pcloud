package tasks

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"text/template"
)

//go:embed env-tmpl
var filesTmpls embed.FS

type activateEnvTask struct {
	basicTask
	env Env
	st  *state
}

func NewActivateEnvTask(env Env, st *state) Task {
	return newSequentialParentTask(
		fmt.Sprintf("Activate new %s instance", env.PCloudEnvName),
		AddNewEnvTask(env, st),
		// TODO(gio): sync dodo-flux
	)
}

func AddNewEnvTask(env Env, st *state) Task {
	t := newLeafTask("Commit initial configuration", func() error {
		ssPublicKeys, err := st.ssClient.GetPublicKeys()
		if err != nil {
			return err
		}
		repoHost := strings.Split(st.ssClient.Addr, ":")[0]
		kust, err := st.repo.ReadKustomization("environments/kustomization.yaml")
		if err != nil {
			return err
		}
		kust.AddResources(env.Name)
		tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
		if err != nil {
			return err
		}
		var knownHosts bytes.Buffer
		for _, key := range ssPublicKeys {
			fmt.Fprintf(&knownHosts, "%s %s\n", repoHost, key)
		}
		for _, tmpl := range tmpls.Templates() {
			dstPath := path.Join("environments", env.Name, tmpl.Name())
			dst, err := st.repo.Writer(dstPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			if err := tmpl.Execute(dst, map[string]string{
				"Name":       env.Name,
				"PrivateKey": base64.StdEncoding.EncodeToString(st.keys.RawPrivateKey()),
				"PublicKey":  base64.StdEncoding.EncodeToString(st.keys.RawAuthorizedKey()),
				"RepoHost":   repoHost,
				"RepoName":   "config",
				"KnownHosts": base64.StdEncoding.EncodeToString(knownHosts.Bytes()),
			}); err != nil {
				return err
			}
		}
		if err := st.repo.WriteKustomization("environments/kustomization.yaml", *kust); err != nil {
			return err
		}
		if err := st.repo.CommitAndPush(fmt.Sprintf("%s: initialize environment", env.Name)); err != nil {
			return err
		}
		return nil
	})
	return &t
}
