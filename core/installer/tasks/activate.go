package tasks

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
)

//go:embed env-tmpl
var filesTmpls embed.FS

func NewActivateEnvTask(env installer.EnvConfig, st *state) Task {
	return newSequentialParentTask(
		"Activate GitOps",
		false,
		AddNewEnvTask(env, st),
		// TODO(gio): sync dodo-flux
	)
}

func AddNewEnvTask(env installer.EnvConfig, st *state) Task {
	t := newLeafTask("Commit initial configuration", func() error {
		ssPublicKeys, err := st.ssClient.GetPublicKeys()
		if err != nil {
			return err
		}
		repoHost := strings.Split(st.ssClient.Address(), ":")[0]
		_, err = st.repo.Do(func(r soft.RepoFS) (string, error) {
			kust, err := soft.ReadKustomization(r, "environments/kustomization.yaml")
			if err != nil {
				return "", err
			}
			kust.AddResources(env.Id)
			tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
			if err != nil {
				return "", err
			}
			var knownHosts bytes.Buffer
			for _, key := range ssPublicKeys {
				fmt.Fprintf(&knownHosts, "%s %s\n", repoHost, key)
			}
			for _, tmpl := range tmpls.Templates() { // TODO(gio): migrate to cue
				dstPath := path.Join("environments", env.Id, tmpl.Name())
				dst, err := r.Writer(dstPath)
				if err != nil {
					return "", err
				}
				defer dst.Close()
				if err := tmpl.Execute(dst, map[string]string{
					"Name":       env.Id,
					"PrivateKey": base64.StdEncoding.EncodeToString(st.keys.RawPrivateKey()),
					"PublicKey":  base64.StdEncoding.EncodeToString(st.keys.RawAuthorizedKey()),
					"RepoHost":   repoHost,
					"RepoName":   "config",
					"KnownHosts": base64.StdEncoding.EncodeToString(knownHosts.Bytes()),
				}); err != nil {
					return "", err
				}
			}
			if err := soft.WriteYaml(r, "environments/kustomization.yaml", kust); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s: initialize environment", env.Id), nil
		})
		return err
	})
	return &t
}
