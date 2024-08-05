package soft

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

var ErrorAlreadyExists = errors.New("already exists")

type Client interface {
	Address() string
	Signer() ssh.Signer
	GetPublicKeys() ([]string, error)
	RepoExists(name string) (bool, error)
	GetRepo(name string) (RepoIO, error)
	GetAllRepos() ([]string, error)
	GetRepoAddress(name string) string
	AddRepository(name string) error
	UserExists(name string) (bool, error)
	FindUser(pubKey string) (string, error)
	GetAllUsers() ([]string, error)
	AddUser(name, pubKey string) error
	RemoveUser(user string) error
	AddPublicKey(user string, pubKey string) error
	RemovePublicKey(user string, pubKey string) error
	GetUserPublicKeys(user string) ([]string, error)
	MakeUserAdmin(name string) error
	AddReadWriteCollaborator(repo, user string) error
	AddReadOnlyCollaborator(repo, user string) error
	AddWebhook(repo, url string, opts ...string) error
}

type realClient struct {
	addr     string
	signer   ssh.Signer
	log      *log.Logger
	pemBytes []byte
}

func NewClient(addr string, clientPrivateKey []byte, log *log.Logger) (Client, error) {
	signer, err := ssh.ParsePrivateKey(clientPrivateKey)
	if err != nil {
		return nil, err
	}
	log.SetPrefix("SOFT-SERVE: ")
	log.Printf("Created signer")
	return &realClient{
		addr,
		signer,
		log,
		clientPrivateKey,
	}, nil
}

type ClientGetter interface {
	Get(addr string, clientPrivateKey []byte, log *log.Logger) (Client, error)
}

type RealClientGetter struct{}

func (c RealClientGetter) Get(addr string, clientPrivateKey []byte, log *log.Logger) (Client, error) {
	var client Client
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

func (ss *realClient) Address() string {
	return ss.addr
}

func (ss *realClient) Signer() ssh.Signer {
	return ss.signer
}

func (ss *realClient) GetAllUsers() ([]string, error) {
	log.Printf("Getting all users")
	out, err := ss.RunCommand("user", "list")
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
}

func (ss *realClient) UserExists(name string) (bool, error) {
	log.Printf("Checking user exists %s", name)
	out, err := ss.RunCommand("user", "list")
	if err != nil {
		return false, err
	}
	return slices.Contains(strings.Fields(out), name), nil
}

func (ss *realClient) FindUser(pubKey string) (string, error) {
	log.Printf("Finding user %s", pubKey)
	pk := strings.Join(strings.Fields(pubKey)[:2], " ")
	out, err := ss.RunCommand("user", "list")
	if err != nil {
		return "", err
	}
	for _, user := range strings.Fields(out) {
		info, err := ss.RunCommand("user", "info", user)
		if err != nil {
			return "", err
		}
		if strings.Contains(info, pk) {
			return user, nil
		}
	}
	return "", nil
}

func (ss *realClient) AddUser(name, pubKey string) error {
	log.Printf("Adding user %s", name)
	if _, err := ss.RunCommand("user", "create", name); err != nil {
		return err
	}
	return ss.AddPublicKey(name, pubKey)
}

func (ss *realClient) RemoveUser(user string) error {
	log.Printf("Removing user: %s\n", user)
	_, err := ss.RunCommand("user", "delete", user)
	return err
}

func (ss *realClient) MakeUserAdmin(name string) error {
	log.Printf("Making user %s admin", name)
	_, err := ss.RunCommand("user", "set-admin", name, "true")
	return err
}

func (ss *realClient) AddPublicKey(user string, pubKey string) error {
	log.Printf("Adding public key: %s %s\n", user, pubKey)
	_, err := ss.RunCommand("user", "add-pubkey", user, pubKey)
	return err
}

func (ss *realClient) RemovePublicKey(user string, pubKey string) error {
	log.Printf("Removing public key: %s %s\n", user, pubKey)
	_, err := ss.RunCommand("user", "remove-pubkey", user, pubKey)
	return err
}

func (ss *realClient) GetUserPublicKeys(user string) ([]string, error) {
	log.Printf("Getting public keys for user: %s\n", user)
	out, err := ss.RunCommand("user", "info", user)
	if err != nil {
		return nil, err
	}
	return extractPublicKeys(out), nil
}

func extractPublicKeys(userInfo string) []string {
	var keys []string
	lines := strings.Split(userInfo, "\n")
	gettingKeys := false
	for _, line := range lines {
		if strings.HasPrefix(line, "Public keys:") {
			gettingKeys = true
			continue
		}
		if gettingKeys {
			if key := CleanKey(line); key != "" {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func CleanKey(key string) string {
	k := strings.TrimSpace(key)
	fields := strings.Fields(k)
	if len(fields) < 2 {
		return k
	}
	return fields[0] + " " + fields[1]
}

func (ss *realClient) RunCommand(args ...string) (string, error) {
	cmd := strings.Join(args, " ")
	log.Printf("Running command %s", cmd)
	client, err := ssh.Dial("tcp", ss.addr, ss.sshClientConfig())
	if err != nil {
		return "", err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var buf strings.Builder
	session.Stdout = &buf
	session.Stderr = os.Stderr
	err = session.Run(cmd)
	return buf.String(), err
}

func (ss *realClient) RepoExists(name string) (bool, error) {
	out, err := ss.RunCommand("repo", "list")
	if err != nil {
		return false, err
	}
	return slices.Contains(strings.Fields(out), name), nil
}

func (ss *realClient) AddRepository(name string) error {
	log.Printf("Adding repository %s", name)
	if ok, err := ss.RepoExists(name); ok {
		return ErrorAlreadyExists
	} else if err != nil {
		return err
	}
	_, err := ss.RunCommand("repo", "create", name)
	return err
}

func (ss *realClient) AddReadWriteCollaborator(repo, user string) error {
	log.Printf("Adding read-write collaborator %s %s", repo, user)
	_, err := ss.RunCommand("repo", "collab", "add", repo, user, "read-write")
	return err
}

func (ss *realClient) AddReadOnlyCollaborator(repo, user string) error {
	log.Printf("Adding read-only collaborator %s %s", repo, user)
	_, err := ss.RunCommand("repo", "collab", "add", repo, user, "read-only")
	return err
}

func (ss *realClient) AddWebhook(repo, url string, opts ...string) error {
	log.Printf("Adding webhook %s %s", repo, url)
	_, err := ss.RunCommand(append(
		[]string{"repo", "webhook", "create", repo, url},
		opts...,
	)...)
	return err
}

type Repository struct {
	*git.Repository
	Addr RepositoryAddress
}

func (ss *realClient) GetRepo(name string) (RepoIO, error) {
	r, err := CloneRepository(RepositoryAddress{ss.addr, name}, ss.signer)
	if err != nil {
		return nil, err
	}
	return NewRepoIO(r, ss.signer)
}

func (ss *realClient) GetAllRepos() ([]string, error) {
	log.Printf("Getting all repos")
	out, err := ss.RunCommand("repo", "list")
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
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
		SingleBranch:    true,
		Depth:           1,
		InsecureSkipTLS: true,
		Progress:        os.Stdout,
	})
	if err != nil && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil, err
	}
	wt, err := c.Worktree()
	if err != nil {
		return nil, err
	}
	sb, err := wt.Submodules()
	if err != nil {
		return nil, err
	}
	if err := sb.Init(); err != nil {
		return nil, err
	}
	if err := sb.Update(&git.SubmoduleUpdateOptions{
		Depth: 1,
	}); err != nil {
		return nil, err
	}
	return &Repository{
		Repository: c,
		Addr:       addr,
	}, nil
}

// TODO(giolekva): dead code
func (ss *realClient) authSSH() gitssh.AuthMethod {
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

func (ss *realClient) authGit() *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		User:   "git",
		Signer: ss.signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				ss.log.Printf("--- %+v\n", ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}

func (ss *realClient) GetPublicKeys() ([]string, error) {
	var ret []string
	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(ss.signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			ret = append(ret, string(ssh.MarshalAuthorizedKey(key)))
			return nil
		},
	}
	client, err := ssh.Dial("tcp", ss.addr, config)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	return ret, nil
}

func (ss *realClient) sshClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(ss.signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// TODO(giolekva): verify server public key
			// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
			fmt.Printf("%s %s %s", hostname, remote, ssh.MarshalAuthorizedKey(key))
			return nil
		},
	}
}

func (ss *realClient) GetRepoAddress(name string) string {
	return fmt.Sprintf("%s/%s", ss.addressGit(), name)
}

func (ss *realClient) addressGit() string {
	return fmt.Sprintf("ssh://%s", ss.addr)
}
