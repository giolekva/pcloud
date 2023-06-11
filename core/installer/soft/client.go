package soft

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	// "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

type Client struct {
	ip     string
	port   int
	Signer ssh.Signer
	log    *log.Logger
}

func NewClient(ip string, port int, clientPrivateKey []byte, log *log.Logger) (*Client, error) {
	signer, err := ssh.ParsePrivateKey(clientPrivateKey)
	if err != nil {
		return nil, err
	}
	log.SetPrefix("SOFT-SERVE: ")
	return &Client{
		ip,
		port,
		signer,
		log,
	}, nil
}

func (ss *Client) AddUser(name, pubKey string) error {
	log.Printf("Adding user %s", name)
	if err := ss.RunCommand(fmt.Sprintf("user create %s", name)); err != nil {
		return err
	}
	return ss.RunCommand(fmt.Sprintf("user add-pubkey %s %s", name, pubKey))
}

func (ss *Client) MakeUserAdmin(name string) error {
	log.Printf("Making user %s admin", name)
	return ss.RunCommand(fmt.Sprintf("user set-admin %s true", name))
}

func (ss *Client) RunCommand(cmd string) error {
	log.Printf("Running command %s", cmd)
	client, err := ssh.Dial("tcp", ss.addressSSH(), ss.sshClientConfig())
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run(cmd)
}

func (ss *Client) AddRepository(name, readme string) error {
	log.Printf("Adding repository %s", name)
	return ss.RunCommand(fmt.Sprintf("repo create %s -d \"%s\"", name, readme))
}

func (ss *Client) AddCollaborator(repo, user string) error {
	log.Printf("Adding collaborator %s %s", repo, user)
	return ss.RunCommand(fmt.Sprintf("repo collab add %s %s", repo, user))
}

func (ss *Client) CreateRepository(name string) error {
	log.Printf("Creating repository %s", name)
	configRepo, err := ss.getConfigRepo()
	if err != nil {
		return err
	}
	wt, err := configRepo.Worktree()
	if err != nil {
		return err
	}
	if err = wt.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/master",
	}); err != nil {
		return err
	}
	f, err := wt.Filesystem.Open("config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()
	configBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	config := make(map[string]interface{})
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	repos := config["repos"].([]interface{})
	repos = append(repos, map[string]interface{}{
		"name":    name,
		"repo":    name,
		"private": true,
		"note":    fmt.Sprintf("PCloud env for %s", name),
	})
	config["repos"] = repos
	configBytes, err = yaml.Marshal(config)
	if err != nil {
		return err
	}
	if err := ss.writeFile(wt, "config.yaml", string(configBytes)); err != nil {
		return err
	}
	if err := ss.Commit(wt, fmt.Sprintf("add-repo: %s", name)); err != nil {
		return err
	}
	return ss.Push(configRepo)
}

func (ss *Client) getConfigRepo() (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             ss.addressGit(),
		Auth:            ss.authGit(),
		RemoteName:      "origin",
		ReferenceName:   "refs/heads/master",
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
}

func (ss *Client) GetRepo(name string) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             fmt.Sprintf("%s/%s", ss.addressGit(), name),
		Auth:            ss.authGit(),
		RemoteName:      "origin",
		ReferenceName:   "refs/heads/master",
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
}

func (ss *Client) repoPathByName(name string) string {
	return fmt.Sprintf("%s/%s", ss.addressGit(), name)
}

func (ss *Client) Commit(wt *git.Worktree, message string) error {
	_, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "pcloud",
			Email: "pcloud@installer",
			When:  time.Now(),
		},
	})
	return err
}

func (ss *Client) Push(repo *git.Repository) error {
	return repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       ss.authGit(),
	})
}

func (ss *Client) writeFile(wt *git.Worktree, path, contents string) error {
	f, err := wt.Filesystem.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write([]byte(contents)); err != nil {
		return err
	}
	_, err = wt.Add(path)
	return err
}

func (ss *Client) CloneRepository(name string) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             ss.repoPathByName(name),
		Auth:            ss.authGit(),
		RemoteName:      "origin",
		InsecureSkipTLS: true,
	})
}

func (ss *Client) authGit() *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		Signer: ss.Signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}

func (ss *Client) GetPublicKey() ([]byte, error) {
	var ret []byte
	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(ss.Signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			ret = ssh.MarshalAuthorizedKey(key)
			return nil
		},
	}
	_, err := ssh.Dial("tcp", ss.addressSSH(), config)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (ss *Client) sshClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(ss.Signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// TODO(giolekva): verify server public key
			// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
			fmt.Printf("%s %s %s", hostname, remote, ssh.MarshalAuthorizedKey(key))
			return nil
		},
	}
}

func (ss *Client) addressGit() string {
	return fmt.Sprintf("ssh://%s:%d", ss.ip, ss.port)
}

func (ss *Client) addressSSH() string {
	return fmt.Sprintf("%s:%d", ss.ip, ss.port)
}
