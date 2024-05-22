package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/soft"
	"github.com/giolekva/pcloud/core/installer/welcome"

	"github.com/spf13/cobra"
)

var dodoAppFlags struct {
	port      int
	sshKey    string
	repoAddr  string
	self      string
	namespace string
	envConfig string
}

func dodoAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dodo-app",
		RunE: dodoAppCmdRun,
	}
	cmd.Flags().IntVar(
		&dodoAppFlags.port,
		"port",
		8080,
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.repoAddr,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.self,
		"self",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.namespace,
		"namespace",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&dodoAppFlags.envConfig,
		"env-config",
		"",
		"",
	)
	return cmd
}

func dodoAppCmdRun(cmd *cobra.Command, args []string) error {
	envConfig, err := os.Open(dodoAppFlags.envConfig)
	if err != nil {
		return err
	}
	defer envConfig.Close()
	var env installer.EnvConfig
	if err := json.NewDecoder(envConfig).Decode(&env); err != nil {
		return err
	}
	sshKey, err := os.ReadFile(dodoAppFlags.sshKey)
	if err != nil {
		return err
	}
	softClient, err := soft.NewClient(dodoAppFlags.repoAddr, sshKey, log.Default())
	if err != nil {
		return err
	}
	jc, err := newJobCreator()
	if err != nil {
		return err
	}
	if err := softClient.AddRepository("app"); err == nil {
		repo, err := softClient.GetRepo("app")
		if err != nil {
			return err
		}
		if err := initRepo(repo); err != nil {
			return err
		}
		if err := welcome.UpdateDodoApp(softClient, dodoAppFlags.namespace, string(sshKey), jc, &env); err != nil {
			return err
		}
		if err := softClient.AddWebhook("app", fmt.Sprintf("http://%s/update", dodoAppFlags.self), "--active=true", "--events=push", "--content-type=json"); err != nil {
			return err
		}
	} else if !errors.Is(err, soft.ErrorAlreadyExists) {
		return err
	}
	s := welcome.NewDodoAppServer(dodoAppFlags.port, string(sshKey), softClient, dodoAppFlags.namespace, jc, env)
	return s.Start()
}

const goMod = `module dodo.app

go 1.18
`

const mainGo = `package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 8080, "Port to listen on")

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from Dodo App!")
}

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
`

const appCue = `app: {
	type: "golang:1.22.0"
	run: "main.go"
	ingress: {
		network: "Private" // or Public
		subdomain: "testapp"
		auth: enabled: false
	}
}
`

func initRepo(repo soft.RepoIO) error {
	return repo.Do(func(fs soft.RepoFS) (string, error) {
		{
			w, err := fs.Writer("go.mod")
			if err != nil {
				return "", err
			}
			defer w.Close()
			fmt.Fprint(w, goMod)
		}
		{
			w, err := fs.Writer("main.go")
			if err != nil {
				return "", err
			}
			defer w.Close()
			fmt.Fprintf(w, "%s", mainGo)
		}
		{
			w, err := fs.Writer("app.cue")
			if err != nil {
				return "", err
			}
			defer w.Close()
			fmt.Fprint(w, appCue)
		}
		return "go web app template", nil
	})
}
