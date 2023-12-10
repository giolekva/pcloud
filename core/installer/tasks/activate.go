package tasks

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/charmbracelet/keygen"

	"github.com/giolekva/pcloud/core/installer"
)

//go:embed env-tmpl
var filesTmpls embed.FS

type activateEnvTask struct {
	basicTask
	env Env
	st  *state
}

func NewActivateEnvTask(env Env, st *state) Task {
	return &activateEnvTask{
		basicTask: basicTask{
			title: fmt.Sprintf("Activate %s environment", env.Name),
		},
		env: env,
		st:  st,
	}
}

func (t *activateEnvTask) Start() {
	ssPublicKeys, err := t.st.ssClient.GetPublicKeys()
	if err != nil {
		t.callDoneListeners(err)
		return
	}
	if err := t.addNewEnv(
		t.st.repo,
		strings.Split(t.st.ssClient.Addr, ":")[0],
		t.st.keys,
		ssPublicKeys,
	); err != nil {
		t.callDoneListeners(err)
		return
	}
	t.callDoneListeners(nil)
}

func (t *activateEnvTask) addNewEnv(
	repoIO installer.RepoIO,
	repoHost string,
	keys *keygen.KeyPair,
	configRepoPublicKeys []string,
) error {
	kust, err := repoIO.ReadKustomization("environments/kustomization.yaml")
	if err != nil {
		return err
	}
	kust.AddResources(t.env.Name)
	tmpls, err := template.ParseFS(filesTmpls, "env-tmpl/*.yaml")
	if err != nil {
		return err
	}
	var knownHosts bytes.Buffer
	for _, key := range configRepoPublicKeys {
		fmt.Fprintf(&knownHosts, "%s %s\n", repoHost, key)
	}
	for _, tmpl := range tmpls.Templates() {
		dstPath := path.Join("environments", t.env.Name, tmpl.Name())
		dst, err := repoIO.Writer(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if err := tmpl.Execute(dst, map[string]string{
			"Name":       t.env.Name,
			"PrivateKey": base64.StdEncoding.EncodeToString(keys.RawPrivateKey()),
			"PublicKey":  base64.StdEncoding.EncodeToString(keys.RawAuthorizedKey()),
			"RepoHost":   repoHost,
			"RepoName":   "config",
			"KnownHosts": base64.StdEncoding.EncodeToString(knownHosts.Bytes()),
		}); err != nil {
			return err
		}
	}
	if err := repoIO.WriteKustomization("environments/kustomization.yaml", *kust); err != nil {
		return err
	}
	if err := repoIO.CommitAndPush(fmt.Sprintf("%s: initialize environment", t.env.Name)); err != nil {
		return err
	}
	return nil
}
