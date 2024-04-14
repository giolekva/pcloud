package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	l           sync.Locker
	port        int
	ready       bool
	cmd         *exec.Cmd
	repoAddr    string
	signer      ssh.Signer
	appDir      string
	runCommands []Command
	self        string
	manager     string
}

func NewServer(port int, repoAddr string, signer ssh.Signer, appDir string, runCommands []Command, self string, manager string) *Server {
	return &Server{
		l:           &sync.Mutex{},
		port:        port,
		ready:       false,
		repoAddr:    repoAddr,
		signer:      signer,
		appDir:      appDir,
		runCommands: runCommands,
		self:        self,
		manager:     manager,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/update", s.handleUpdate)
	http.HandleFunc("/ready", s.handleReady)
	if err := s.run(); err != nil {
		return err
	}
	go s.pingManager()
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.l.Lock()
	defer s.l.Unlock()
	if s.ready {
		fmt.Fprintln(w, "ok")
	} else {
		http.Error(w, "not ready", http.StatusInternalServerError)
	}
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	fmt.Println("update")
	s.l.Lock()
	s.ready = false
	s.l.Unlock()
	if s.cmd != nil {
		err := s.cmd.Process.Kill()
		s.cmd.Wait()
		s.cmd = nil
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := os.RemoveAll(s.appDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.l.Lock()
	s.ready = true
	s.l.Unlock()
}

func (s *Server) run() error {
	if err := CloneRepository(s.repoAddr, s.signer, s.appDir); err != nil {
		return err
	}
	for i, c := range s.runCommands {
		args := []string{c.Bin}
		args = append(args, c.Args...)
		cmd := &exec.Cmd{
			Dir:    *appDir,
			Path:   c.Bin,
			Args:   args,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		fmt.Printf("Running: %s\n", c.Bin)
		if i < len(s.runCommands)-1 {
			if err := cmd.Run(); err != nil {
				return err
			}
		} else {
			if err := cmd.Start(); err != nil {
				return err
			}
			s.cmd = cmd
		}
	}
	return nil
}

type pingReq struct {
	Address string `json:"address"`
}

func (s *Server) pingManager() {
	defer func() {
		go func() {
			time.Sleep(5 * time.Second)
			s.pingManager()
		}()
	}()
	buf, err := json.Marshal(pingReq{s.self})
	if err != nil {
		return
	}
	http.Post(s.manager, "application/json", bytes.NewReader(buf))
}
