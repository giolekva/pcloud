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
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/crypto/ssh"
	"sigs.k8s.io/yaml"
)

type Client struct {
	ip     string
	port   int
	signer ssh.Signer
	log    *log.Logger
}

func NewClient(ip string, port int, clientPrivateKey []byte, log *log.Logger) (*Client, error) {
	signer, err := ssh.ParsePrivateKey(clientPrivateKey)
	if err != nil {
		return nil, err
	}
	log.SetPrefix("SOFT-SERVE")
	return &Client{
		ip,
		port,
		signer,
		log,
	}, nil
}

func (ss *Client) UpdateConfig(config Config, reason string) error {
	log.Print("Updating configuration")
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	repo, err := ss.cloneRepository("config")
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := ss.writeFile(wt, "config.yaml", string(configBytes)); err != nil {
		return err
	}
	if err := ss.commit(wt, reason); err != nil {
		return nil
	}
	return ss.push(repo)
}

func (ss *Client) ReloadConfig() error {
	log.Print("Reloading configuration")
	client, err := ssh.Dial("tcp", ss.addressSSH(), ss.sshClientConfig())
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run("reload")
}

func (ss *Client) AddRepository(name, readme string) error {
	log.Printf("Adding repository %s", name)
	repo, err := git.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := ss.writeFile(wt, "README.md", readme); err != nil {
		return err
	}
	if err := ss.commit(wt, "init"); err != nil {
		return err
	}
	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "soft",
		URLs: []string{ss.repoPathByName(name)},
	}); err != nil {
		return err
	}
	return ss.push(repo)
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
	fmt.Println("aaaa")
	b, _ := configRepo.Branches()
	b.ForEach(func(r *plumbing.Reference) error {
		fmt.Println(r.Name())
		return nil
	})
	if err = wt.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/master",
	}); err != nil {
		return err
	}
	fmt.Println("bbb")
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
	if err := ss.commit(wt, fmt.Sprintf("add-repo: %s", name)); err != nil {
		return err
	}
	return ss.push(configRepo)
}

func (ss *Client) getConfigRepo() (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             ss.addressGit(),
		Auth:            ss.authGit(),
		RemoteName:      "soft",
		ReferenceName:   "refs/heads/master",
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
}

func (ss *Client) repoPathByName(name string) string {
	return fmt.Sprintf("%s/%s", ss.addressGit(), name)
}

func (ss *Client) commit(wt *git.Worktree, message string) error {
	_, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "pcloud",
			Email: "pcloud@installer",
			When:  time.Now(),
		},
	})
	return err
}

func (ss *Client) push(repo *git.Repository) error {
	return repo.Push(&git.PushOptions{
		RemoteName: "soft",
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

func (ss *Client) cloneRepository(name string) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             ss.repoPathByName(name),
		Auth:            ss.authGit(),
		RemoteName:      "soft",
		InsecureSkipTLS: true,
	})
}

func (ss *Client) authGit() *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		Signer: ss.signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}

func (ss *Client) sshClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(ss.signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// TODO(giolekva): verify server public key
			// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
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
