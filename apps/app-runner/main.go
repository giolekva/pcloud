package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

var port = flag.Int("port", 3000, "Port to listen on")
var repoAddr = flag.String("repo-addr", "", "Git repository address")
var sshKey = flag.String("ssh-key", "", "Private SSH key to access Git repository")
var appDir = flag.String("app-dir", "", "Path to store application repository locally")
var runCfg = flag.String("run-cfg", "", "Run configuration")
var manager = flag.String("manager", "", "Address of the manager")

type Command struct {
	Bin  string   `json:"bin"`
	Args []string `json:"args"`
}

func CloneRepository(addr string, signer ssh.Signer, path string) error {
	c, err := git.Clone(memory.NewStorage(), osfs.New(path, osfs.WithBoundOS()), &git.CloneOptions{
		URL: addr,
		Auth: &gitssh.PublicKeys{
			User:   "git",
			Signer: signer,
			HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
				HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
					// TODO(giolekva): verify server public key
					fmt.Printf("--- %+v\n", ssh.MarshalAuthorizedKey(key))
					return nil
				},
			},
		},
		RemoteName:      "origin",
		ReferenceName:   "refs/heads/master",
		SingleBranch:    true,
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
	if err != nil {
		return err
	}
	wt, err := c.Worktree()
	if err != nil {
		return err
	}
	sb, err := wt.Submodules()
	if err != nil {
		return err
	}
	if err := sb.Init(); err != nil {
		return err
	}
	if err := sb.Update(&git.SubmoduleUpdateOptions{
		Depth: 1,
	}); err != nil {
		return err
	}
	return err
}

func main() {
	flag.Parse()
	self, ok := os.LookupEnv("SELF_IP")
	if !ok {
		panic("no SELF_IP")
	}
	key, err := os.ReadFile(*sshKey)
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	if err := CloneRepository(*repoAddr, signer, *appDir); err != nil {
		panic(err)
	}
	r, err := os.Open(*runCfg)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	var cmds []Command
	if err := json.NewDecoder(r).Decode(&cmds); err != nil {
		panic(err)
	}
	s := NewServer(*port, *repoAddr, signer, *appDir, cmds, self, *manager)
	s.Start()
}
