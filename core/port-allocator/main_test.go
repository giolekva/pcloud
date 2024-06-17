package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
)

type fakeClient struct {
	contents map[string]any
}

func (c *fakeClient) ReadRelease() (map[string]any, error) {
	return c.contents, nil
}

func (c *fakeClient) WriteRelease(rel map[string]any, meta string) error {
	c.contents = rel
	return nil
}

func TestAllocateSucceeds(t *testing.T) {
	c := &fakeClient{map[string]any{
		"spec": map[string]any{
			"values": map[string]any{},
		},
	}}
	s := newServer(8080, c) // TODO(gio): run using unix socket
	go func() {
		s.Start()
	}()
	defer s.Close()
	var buf bytes.Buffer
	req := allocateReq{
		Protocol:      "TCP",
		SourcePort:    22,
		TargetService: "foo",
		TargetPort:    2222,
	}
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post("http://localhost:8080/api/allocate", "application/json", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(os.Stdout, resp.Body)
		t.Fatalf("Expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
	expected := map[string]any{
		"spec": map[string]any{
			"values": map[string]any{
				"tcp": map[string]any{
					"22": "foo:2222",
				},
				"udp": map[string]any{},
			},
		},
	}
	if !reflect.DeepEqual(expected, c.contents) {
		t.Fatalf("Expected %v, got %v", expected, c.contents)
	}
}

func TestAllocateConflicts(t *testing.T) {
	c := &fakeClient{map[string]any{
		"spec": map[string]any{
			"values": map[string]any{
				"tcp": map[string]any{
					"22": "foo:2222",
				},
			},
		},
	}}
	s := newServer(8080, c) // TODO(gio): run using unix socket
	go func() {
		s.Start()
	}()
	defer s.Close()
	var buf bytes.Buffer
	req := allocateReq{
		Protocol:      "TCP",
		SourcePort:    22,
		TargetService: "foo",
		TargetPort:    2222,
	}
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post("http://localhost:8080/api/allocate", "application/json", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		io.Copy(os.Stdout, resp.Body)
		t.Fatalf("Expected %d, got %d", http.StatusConflict, resp.StatusCode)
	}
}

func TestAllocate80Taken(t *testing.T) {
	c := &fakeClient{map[string]any{
		"spec": map[string]any{
			"values": map[string]any{},
		},
	}}
	s := newServer(8080, c) // TODO(gio): run using unix socket
	go func() {
		s.Start()
	}()
	defer s.Close()
	var buf bytes.Buffer
	req := allocateReq{
		Protocol:      "TCP",
		SourcePort:    80,
		TargetService: "foo",
		TargetPort:    2222,
	}
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post("http://localhost:8080/api/allocate", "application/json", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		io.Copy(os.Stdout, resp.Body)
		t.Fatalf("Expected %d, got %d", http.StatusConflict, resp.StatusCode)
	}
}

func TestAllocate443Taken(t *testing.T) {
	c := &fakeClient{map[string]any{
		"spec": map[string]any{
			"values": map[string]any{},
		},
	}}
	s := newServer(8080, c) // TODO(gio): run using unix socket
	go func() {
		s.Start()
	}()
	defer s.Close()
	var buf bytes.Buffer
	req := allocateReq{
		Protocol:      "TCP",
		SourcePort:    443,
		TargetService: "foo",
		TargetPort:    2222,
	}
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post("http://localhost:8080/api/allocate", "application/json", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict {
		io.Copy(os.Stdout, resp.Body)
		t.Fatalf("Expected %d, got %d", http.StatusConflict, resp.StatusCode)
	}
}

func TestGenerateSecret(t *testing.T) {
	secret, err := generateSecret()
	if err != nil {
		t.Fatalf("Error generating secret: %v", err)
	}
	t.Logf("Generated secret: %s", secret)
}

func TestReservePort(t *testing.T) {
	pm := map[string]struct{}{
		"10000": {},
	}
	reserve := make(map[int]string)
	for i := start; i <= end; i++ {
		reserve[i] = "reserved"
	}
	_, err := reservePort(pm, reserve)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}
