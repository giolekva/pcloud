package soft

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"net/netip"
	"os"
	"regexp"
	"strings"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Client struct {
	Addr     netip.AddrPort
	Signer   ssh.Signer
	log      *log.Logger
	pemBytes []byte
}

func NewClient(addr netip.AddrPort, clientPrivateKey []byte, log *log.Logger) (*Client, error) {
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
	log.Printf("Adding public key: %s %s\n", user, pubKey)
	return ss.RunCommand("user", "remove-pubkey", user, pubKey)
}

func (ss *Client) RunCommand(args ...string) error {
	cmd := strings.Join(args, " ")
	log.Printf("Running command %s", cmd)
	client, err := ssh.Dial("tcp", ss.Addr.String(), ss.sshClientConfig())
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	return session.Run(cmd)
}

func (ss *Client) AddRepository(name, readme string) error {
	log.Printf("Adding repository %s", name)
	return ss.RunCommand("repo", "create", name, "-d", fmt.Sprintf("\"%s\"", readme))
}

func (ss *Client) AddCollaborator(repo, user string) error {
	log.Printf("Adding collaborator %s %s", repo, user)
	return ss.RunCommand("repo", "collab", "add", repo, user)
}

type Repository struct {
	*git.Repository
	Addr RepositoryAddress
}

func (ss *Client) GetRepo(name string) (*Repository, error) {
	return CloneRepository(RepositoryAddress{ss.Addr, name}, ss.Signer)
}

type RepositoryAddress struct {
	Addr netip.AddrPort
	Name string
}

func ParseRepositoryAddress(addr string) (RepositoryAddress, error) {
	items := regexp.MustCompile(`ssh://.*)/(.*)`).FindStringSubmatch(addr)
	if len(items) != 2 {
		return RepositoryAddress{}, fmt.Errorf("Invalid address")
	}
	ipPort, err := netip.ParseAddrPort(items[1])
	if err != nil {
		return RepositoryAddress{}, err
	}
	return RepositoryAddress{ipPort, items[2]}, nil
}

func (r RepositoryAddress) FullAddress() string {
	return fmt.Sprintf("ssh://%s/%s", r.Addr, r.Name)
}

func CloneRepository(addr RepositoryAddress, signer ssh.Signer) (*Repository, error) {
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
	if err != nil {
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
	_, err := ssh.Dial("tcp", ss.Addr.String(), config)
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

func (ss *Client) GetRepoAddress(name string) string {
	return fmt.Sprintf("%s/%s", ss.addressGit(), name)
}

func (ss *Client) addressGit() string {
	return fmt.Sprintf("ssh://%s", ss.Addr)
}
