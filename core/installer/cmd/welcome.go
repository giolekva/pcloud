package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"os"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/giolekva/pcloud/core/installer/welcome"
)

var welcomeFlags struct {
	repo   string
	sshKey string
	port   int
}

func welcomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "welcome",
		RunE: welcomeCmdRun,
	}
	cmd.Flags().StringVar(
		&welcomeFlags.repo,
		"repo-addr",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&welcomeFlags.sshKey,
		"ssh-key",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&welcomeFlags.port,
		"port",
		8080,
		"",
	)
	return cmd
}

func welcomeCmdRun(cmd *cobra.Command, args []string) error {
	sshKey, err := os.ReadFile(welcomeFlags.sshKey)
	if err != nil {
		return err
	}
	auth := authSSH(sshKey)
	repo, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             welcomeFlags.repo,
		Auth:            auth,
		RemoteName:      "origin",
		ReferenceName:   "refs/heads/master",
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
	nsCreator, err := newNSCreator()
	if err != nil {
		return err
	}
	s := welcome.NewServer(
		welcomeFlags.port,
		installer.NewRepoIO(repo, auth.Signer),
		nsCreator,
	)
	s.Start()
	return nil
}

func authSSH(pemBytes []byte) *gitssh.PublicKeys {
	a, err := gitssh.NewPublicKeys("git", pemBytes, "")
	if err != nil {
		panic(err)
	}
	a.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// TODO(giolekva): verify server public key
		fmt.Printf("--- %+v\n", ssh.MarshalAuthorizedKey(key))
		return nil
	}
	return a
}
