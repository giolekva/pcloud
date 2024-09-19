package installer

import (
	"testing"

	"cuelang.org/go/cue"
)

func TestFindPortFields(t *testing.T) {
	scm := structSchema{
		"a",
		[]Field{
			Field{"x", basicSchema{"x", KindString, false, nil}},
			Field{"y", basicSchema{"y", KindInt, false, nil}},
			Field{"z", basicSchema{"z", KindPort, false, nil}},
			Field{
				"w",
				structSchema{
					"w",
					[]Field{
						Field{"x", basicSchema{"x", KindString, false, nil}},
						Field{"y", basicSchema{"y", KindInt, false, nil}},
						Field{"z", basicSchema{"z", KindPort, false, nil}},
					},
					false,
				},
			},
		},
		false,
	}
	p := findPortFields(scm)
	if len(p) != 2 {
		t.Fatalf("expected two port fields, %v", p)
	}
	if p[0] != "z" || p[1] != "w.z" {
		t.Fatalf("expected 'z' and 'w.z' port fields, %v", p)
	}
}

const isNotNetwork = `
input: {
	repoAddr: string
	repoPublicAddr: string
	managerAddr: string
	appId: string
	branch: string
	sshPrivateKey: string
	username?: string
	cluster: #Cluster
	username?: string | *"test"
	vpnAuthKey: string  @role(VPNAuthKey) @usernameField(username)
}

#Cluster: {
	name: string
	kubeconfig: string
	ingressClassName: string
}
`

func TestIsNotNetwork(t *testing.T) {
	v, err := ParseCueAppConfig(CueAppData{"/test.cue": []byte(isNotNetwork)})
	if err != nil {
		t.Fatal(err)
	}
	if isNetwork(v.LookupPath(cue.ParsePath("input"))) {
		t.Fatal("not really network")
	}
}

const inputIsNetwork = `
input: {
	name: string
	ingressClass: string
	certificateIssuer: string | *""
	domain: string
	allocatePortAddr: string
	reservePortAddr: string
	deallocatePortAddr: string
}
`

func TestIsNetwork(t *testing.T) {
	v, err := ParseCueAppConfig(CueAppData{"/test.cue": []byte(inputIsNetwork)})
	if err != nil {
		t.Fatal(err)
	}
	if !isNetwork(v.LookupPath(cue.ParsePath("input"))) {
		t.Fatal("is network")
	}
}
