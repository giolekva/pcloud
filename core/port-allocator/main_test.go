package main

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/giolekva/pcloud/core/installer/soft"

	"github.com/go-git/go-billy/v5/memfs"
)

func fakeSecretGenerator(secret string) SecretGenerator {
	return func() (string, error) {
		return secret, nil
	}
}

func TestAllocateSucceeds(t *testing.T) {
	ingressPath := "/ingress.yaml"
	fs := memfs.New()
	repo := soft.NewMockRepoIO(soft.NewBillyRepoFS(fs), "foo.bar", t)
	if err := soft.WriteYaml(repo, ingressPath, map[string]any{
		"spec": map[string]any{
			"values": map[string]any{
				"controller": map[string]any{
					"service": map[string]any{
						"type": "ClusterIP",
					},
				},
				"tcp": map[string]any{},
				"udp": map[string]any{},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	c, err := newRepoClient(repo, ingressPath, 5, 10, fakeSecretGenerator("test"))
	if err != nil {
		t.Fatal(err)
	}
	tcp := map[string]any{}
	udp := map[string]any{}
	expected := map[string]any{
		"spec": map[string]any{
			"values": map[string]any{
				"controller": map[string]any{
					"service": map[string]any{
						"type": "ClusterIP",
					},
				},
				"tcp": tcp,
				"udp": udp,
			},
		},
	}
	for i := 0; i < 500; i++ {
		for _, protocol := range []string{"tcp", "udp"} {
			port, secret, err := c.ReservePort()
			if err != nil {
				t.Fatal(err)
			}
			target := fmt.Sprintf("%s/bar:%d", protocol, port)
			if err := c.AddPortForwarding("tcp", port, secret, target); err != nil {
				t.Fatal(err)
			}
			tcp[strconv.Itoa(port)] = target
		}
	}
	var actual map[string]any
	if err := soft.ReadYaml(repo, ingressPath, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

func TestGenerateSecret(t *testing.T) {
	secret, err := generateSecret()
	if err != nil {
		t.Fatalf("Error generating secret: %v", err)
	}
	t.Logf("Generated secret: %s", secret)
}
