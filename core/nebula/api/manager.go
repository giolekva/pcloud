package main

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/jinzhu/copier"
	"github.com/slackhq/nebula/cert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	nebulav1 "github.com/giolekva/pcloud/core/nebula/controller/apis/nebula/v1"
	clientset "github.com/giolekva/pcloud/core/nebula/controller/generated/clientset/versioned"
)

type nebulaCA struct {
	Name      string
	Namespace string
	Nodes     []nebulaNode
}

type nebulaNode struct {
	Name      string
	Namespace string
	IP        string
}

type Manager struct {
	kubeClient   kubernetes.Interface
	nebulaClient clientset.Interface
	namespace    string
	caName       string
	cfgTmpl      map[string]interface{}
}

func (m *Manager) ListAll() ([]*nebulaCA, error) {
	ret := make([]*nebulaCA, 0)
	cas, err := m.nebulaClient.LekvaV1().NebulaCAs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, ca := range cas.Items {
		ret = append(ret, &nebulaCA{
			Name:      ca.Name,
			Namespace: ca.Namespace,
			Nodes:     make([]nebulaNode, 0),
		})
	}
	nodes, err := m.nebulaClient.LekvaV1().NebulaNodes("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		for _, ca := range ret {
			if ca.Name == node.Spec.CAName {
				ca.Nodes = append(ca.Nodes, nebulaNode{
					Name:      node.Name,
					Namespace: node.Namespace,
					IP:        node.Spec.IPCidr,
				})
			}
		}
	}
	return ret, nil
}

func (m *Manager) CreateNode(namespace, name, caNamespace, caName, ipCidr, pubKey string, encPubKey []byte) (string, string, error) {
	node := &nebulav1.NebulaNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nebulav1.NebulaNodeSpec{
			CAName:      caName,
			CANamespace: caNamespace,
			IPCidr:      ipCidr,
			PubKey:      pubKey,
			SecretName:  fmt.Sprintf("%s-cert", name),
		},
	}
	if encPubKey != nil {
		node.Spec.EncPubKey = base64.StdEncoding.EncodeToString(encPubKey)
	}
	node, err := m.nebulaClient.LekvaV1().NebulaNodes(namespace).Create(context.TODO(), node, metav1.CreateOptions{})
	if err != nil {
		return "", "", err
	}
	return node.Namespace, node.Name, nil
}

func (m *Manager) GetNodeConfig(namespace, name string) (map[string]interface{}, error) {
	secret, err := m.getNodeSecret(namespace, name)
	if err != nil {
		return nil, err
	}
	var c map[string]interface{}
	if err := copier.CopyWithOption(&c, m.cfgTmpl, copier.Option{DeepCopy: true}); err != nil {
		return nil, err
	}
	var pki map[string]interface{}
	var ok bool
	if pki, ok = c["pki"].(map[string]interface{}); !ok {
		panic("Should not reach")
	}
	pki["ca"] = string(secret.Data["ca.crt"])
	pki["cert"] = string(secret.Data["host.crt"])
	return c, nil
}

func (m *Manager) GetNodeCertQR(namespace, name string) ([]byte, error) {
	secret, err := m.getNodeSecret(namespace, name)
	if err != nil {
		return nil, err
	}
	return secret.Data["host.png"], nil
}

func (m *Manager) GetCACertQR(namespace, name string) ([]byte, error) {
	secret, err := m.getCASecret(namespace, name)
	if err != nil {
		return nil, err
	}
	return secret.Data["ca.png"], nil
}

func (m *Manager) Sign(message []byte) ([]byte, error) {
	secret, err := m.getCASecret(m.namespace, m.caName)
	if err != nil {
		return nil, err
	}
	edPriv, _, err := cert.UnmarshalEd25519PrivateKey(secret.Data["ca.key"])
	if err != nil {
		return nil, err
	}
	return edPriv.Sign(rand.Reader, message, crypto.Hash(0))
}

func (m *Manager) VerifySignature(message, signature []byte) (bool, error) {
	secret, err := m.getCASecret(m.namespace, m.caName)
	if err != nil {
		return false, err
	}
	edPriv, _, err := cert.UnmarshalEd25519PrivateKey(secret.Data["ca.key"])
	if err != nil {
		return false, err
	}
	return ed25519.Verify(edPriv.Public().(ed25519.PublicKey), message, signature), nil
}

func (m *Manager) getCASecret(namespace, name string) (*corev1.Secret, error) {
	ca, err := m.nebulaClient.LekvaV1().NebulaCAs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return m.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), ca.Spec.SecretName, metav1.GetOptions{})
}

func (m *Manager) getNodeSecret(namespace, name string) (*corev1.Secret, error) {
	node, err := m.nebulaClient.LekvaV1().NebulaNodes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return m.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), node.Spec.SecretName, metav1.GetOptions{})
}

func (m *Manager) GetNodeEncryptionPublicKey(namespace, name string) (*rsa.PublicKey, error) {
	node, err := m.nebulaClient.LekvaV1().NebulaNodes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	k, err := base64.StdEncoding.DecodeString(node.Spec.EncPubKey)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS1PublicKey(k)
}
