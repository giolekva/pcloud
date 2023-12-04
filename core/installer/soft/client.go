package soft

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Client struct {
	Addr     string
	Signer   ssh.Signer
	log      *log.Logger
	pemBytes []byte
}

func NewClient(addr string, clientPrivateKey []byte, log *log.Logger) (*Client, error) {
	signer, err := ssh.ParsePrivateKey(clientPrivateKey)
	if err != nil {
		return nil, err
	}
	log.SetPrefix("SOFT-SERVE: ")
	log.Printf("Created signer")
	return &Client{
		addr,
		signer,
		log,
		clientPrivateKey,
	}, nil
}

func WaitForClient(addr string, clientPrivateKey []byte, log *log.Logger) (*Client, error) {
	var client *Client
	err := backoff.RetryNotify(func() error {
		var err error
		client, err = NewClient(addr, clientPrivateKey, log)
		if err != nil {
			return err
		}
		if _, err := client.GetPublicKeys(); err != nil {
			return err
		}
		return nil
	}, backoff.NewConstantBackOff(5*time.Second), func(err error, _ time.Duration) {
		log.Printf("Failed to create client:  %s\n", err.Error())
	})
	return client, err
}

func (ss *Client) AddUser(name, pubKey string) error {
	log.Printf("Adding user %s", name)
	if err := ss.RunCommand("user", "create", name); err != nil {
		return err
	}
	return ss.AddPublicKey(name, pubKey)
}

func (ss *Client) MakeUserAdmin(name string) error {
	log.Printf("Making user %s admin", name)
	return ss.RunCommand("user", "set-admin", name, "true")
}

func (ss *Client) AddPublicKey(user string, pubKey string) error {
	log.Printf("Adding public key: %s %s\n", user, pubKey)
	return ss.RunCommand("user", "add-pubkey", user, pubKey)
}

func (ss *Client) RemovePublicKey(user string, pubKey string) error {
	log.Printf("Removing public key: %s %s\n", user, pubKey)
	return ss.RunCommand("user", "remove-pubkey", user, pubKey)
}

func (ss *Client) RunCommand(args ...string) error {
	cmd := strings.Join(args, " ")
	log.Printf("Running command %s", cmd)
	client, err := ssh.Dial("tcp", ss.Addr, ss.sshClientConfig())
	if err != nil {
		return err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	return session.Run(cmd)
}

func (ss *Client) AddRepository(name string) error {
	log.Printf("Adding repository %s", name)
	return ss.RunCommand("repo", "create", name)
}

func (ss *Client) AddReadWriteCollaborator(repo, user string) error {
	log.Printf("Adding read-write collaborator %s %s", repo, user)
	return ss.RunCommand("repo", "collab", "add", repo, user, "read-write")
}

func (ss *Client) AddReadOnlyCollaborator(repo, user string) error {
	log.Printf("Adding read-only collaborator %s %s", repo, user)
	return ss.RunCommand("repo", "collab", "add", repo, user, "read-only")
}

type Repository struct {
	*git.Repository
	Addr RepositoryAddress
}

func (ss *Client) GetRepo(name string) (*Repository, error) {
	return CloneRepository(RepositoryAddress{ss.Addr, name}, ss.Signer)
}

type RepositoryAddress struct {
	Addr string
	Name string
}

func ParseRepositoryAddress(addr string) (RepositoryAddress, error) {
	items := regexp.MustCompile(`ssh://(.*)/(.*)`).FindStringSubmatch(addr)
	if len(items) != 3 {
		return RepositoryAddress{}, fmt.Errorf("Invalid address")
	}
	return RepositoryAddress{items[1], items[2]}, nil
}

func (r RepositoryAddress) FullAddress() string {
	return fmt.Sprintf("ssh://%s/%s", r.Addr, r.Name)
}

func CloneRepository(addr RepositoryAddress, signer ssh.Signer) (*Repository, error) {
	fmt.Printf("Cloning repository: %s %s\n", addr.Addr, addr.Name)
	c, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL: addr.FullAddress(),
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
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
	if err != nil && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil, err
	}
	return &Repository{
		Repository: c,
		Addr:       addr,
	}, nil
}

// TODO(giolekva): dead code
func (ss *Client) authSSH() gitssh.AuthMethod {
	a, err := gitssh.NewPublicKeys("git", ss.pemBytes, "")
	if err != nil {
		panic(err)
	}
	a.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// TODO(giolekva): verify server public key
		ss.log.Printf("--- %+v\n", ssh.MarshalAuthorizedKey(key))
		return nil
	}
	return a
	// return &gitssh.PublicKeys{
	// 	User:   "git",
	// 	Signer: ss.Signer,
	// 	HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
	// 		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	// 			// TODO(giolekva): verify server public key
	// 			ss.log.Printf("--- %+v\n", ssh.MarshalAuthorizedKey(key))
	// 			return nil
	// 		},
	// 	},
	// }
}

func (ss *Client) authGit() *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		User:   "git",
		Signer: ss.Signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				ss.log.Printf("--- %+v\n", ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}

func (ss *Client) GetPublicKeys() ([]string, error) {
	var ret []string
	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(ss.Signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			ret = append(ret, string(ssh.MarshalAuthorizedKey(key)))
			return nil
		},
	}
	client, err := ssh.Dial("tcp", ss.Addr, config)
	if err != nil {
		return nil, err
	}
	defer client.Close()
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

func (ss *Client) GetRepoAddress(name string) string {
	return fmt.Sprintf("%s/%s", ss.addressGit(), name)
}

func (ss *Client) addressGit() string {
	return fmt.Sprintf("ssh://%s", ss.Addr)
}
