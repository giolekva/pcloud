package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var port = flag.Int("port", 8080, "Port to listen on.")
var maddyConfig = flag.String("maddy-config", "", "Path to the Maddy configuration file.")
var exportDKIM = flag.String("export-dkim", "", "Path to the dkim dns configuration to expose.")

//go:embed templates/*
var tmpls embed.FS

type Templates struct {
	Index *template.Template
}

func ParseTemplates(fs embed.FS) (*Templates, error) {
	index, err := template.ParseFS(fs, "templates/index.html")
	if err != nil {
		return nil, err
	}
	return &Templates{index}, nil
}

type MaddyManager struct {
	configPath string
}

func (m MaddyManager) ListAccounts() ([]string, error) {
	cmd := exec.Command("maddyctl", "-config", m.configPath, "creds", "list")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	accts := make([]string, 0)
	for scanner.Scan() {
		acct := scanner.Text()
		if len(acct) == 0 {
			continue
		}
		accts = append(accts, acct)
	}
	return accts, nil

}

func (m MaddyManager) CreateAccount(username, password string) error {
	cmd := exec.Command("maddyctl", "-config", m.configPath, "creds", "create", username)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, password)
	}()
	if err := cmd.Wait(); err != nil {
		return err
	}
	// Create IMAP
	cmd = exec.Command("maddyctl", "-config", m.configPath, "imap-acct", "create", username)
	return cmd.Run()
}

type MaddyHandler struct {
	mgr   MaddyManager
	tmpls *Templates
}

func (h *MaddyHandler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accts, err := h.mgr.ListAccounts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.tmpls.Index.Execute(w, accts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *MaddyHandler) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.mgr.CreateAccount(r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *MaddyHandler) handleDKIM(w http.ResponseWriter, r *http.Request) {
	d, err := os.Open(*exportDKIM)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer d.Close()
	if _, err := io.Copy(w, d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	flag.Parse()
	t, err := ParseTemplates(tmpls)
	if err != nil {
		log.Fatal(err)
	}
	mgr := MaddyManager{
		configPath: *maddyConfig,
	}
	handler := MaddyHandler{
		mgr:   mgr,
		tmpls: t,
	}
	http.HandleFunc("/", handler.handleListAccounts)
	http.HandleFunc("/create", handler.handleCreateAccount)
	if *exportDKIM != "" {
		http.HandleFunc("/dkim", handler.handleDKIM)
	}
	fmt.Printf("Starting HTTP server on port: %d\n", *port)
	fmt.Printf("Maddy config: %s\n", *maddyConfig)
	if cfg, err := ioutil.ReadFile(*maddyConfig); err != nil {
		log.Fatal(err)
	} else {
		log.Print(string(cfg))
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

}
