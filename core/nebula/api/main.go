package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"sigs.k8s.io/yaml"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/giolekva/pcloud/core/nebula/controller/generated/clientset/versioned"
)

var port = flag.Int("port", 8080, "Port to listen on.")
var kubeConfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
var masterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
var namespace = flag.String("namespace", "", "Namespace where Nebula CA and Node secrets are stored.")
var caName = flag.String("ca-name", "", "Name of the Nebula CA.")
var configTmpl = flag.String("config-tmpl", "", "Path to the lighthouse configuration template file.")

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

type Handler struct {
	mgr   Manager
	tmpls *Templates
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	cas, err := h.mgr.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.tmpls.Index.Execute(w, cas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) handleNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	qr, err := h.mgr.GetNodeCertQR(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "img/png")
	w.Write(qr)
}

func (h *Handler) handleCA(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	qr, err := h.mgr.GetCACertQR(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "img/png")
	w.Write(qr)
}

type signReq struct {
	Message []byte `json:"message"`
}

type signResp struct {
	Signature []byte `json:"signature"`
}

func (h *Handler) sign(w http.ResponseWriter, r *http.Request) {
	var req signReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	signature, err := h.mgr.Sign(req.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	resp := signResp{
		signature,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type joinReq struct {
	Message   []byte `json:"message"`
	Signature []byte `json:"signature"`
	Name      string `json:"name"`
	PublicKey []byte `json:"public_key"`
	IPCidr    string `json:"ip_cidr"`
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	var req joinReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	valid, err := h.mgr.VerifySignature(req.Message, req.Signature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, "Signature could not be verified", http.StatusBadRequest)
		return
	}
	_, _, err = h.mgr.CreateNode(
		*namespace,
		req.Name,
		*namespace,
		*caName,
		req.IPCidr,
		string(req.PublicKey),
		nil,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for {
		time.Sleep(1 * time.Second)
		cfg, err := h.mgr.GetNodeConfig(*namespace, req.Name)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		cfgBytes, err := yaml.Marshal(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfgB64 := base64.StdEncoding.EncodeToString(cfgBytes)
		if _, err := fmt.Fprint(w, cfgB64); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		break
	}
}

type getResp struct {
	Key   []byte `json:"key"`
	Nonce []byte `json:"nonce"`
	Data  []byte `json:"data"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	fmt.Println("##### GET")
	vars := mux.Vars(r)
	pubKey, err := h.mgr.GetNodeEncryptionPublicKey(*namespace, vars["name"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("Got key")
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	for {
		time.Sleep(1 * time.Second)
		cfg, err := h.mgr.GetNodeConfig(*namespace, vars["name"])
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		cfgBytes, err := yaml.Marshal(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfgEnc := aesgcm.Seal(nil, nonce, cfgBytes, nil)
		keyEnc, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, key, []byte(""))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nonceEnc, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, nonce, []byte(""))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := getResp{
			Key:   keyEnc,
			Nonce: nonceEnc,
			Data:  cfgEnc,
		}
		if err := json.NewEncoder(w).Encode(&resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		break
	}
}

type approveReq struct {
	EncPublicKey []byte `json:"enc_public_key"`
	Name         string `json:"name"`
	NetPublicKey []byte `json:"net_public_key"`
	IPCidr       string `json:"ip_cidr"`
}

type approveResp struct {
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	var req approveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("---- APPROVE")
	fmt.Printf("%#v\n", req)
	_, _, err := h.mgr.CreateNode(
		*namespace,
		req.Name,
		*namespace,
		*caName,
		req.IPCidr,
		string(req.NetPublicKey),
		req.EncPublicKey,
	)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func loadConfigTemplate(path string) (map[string]interface{}, error) {
	tmpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(tmpl, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func main() {
	flag.Parse()
	cfgTmpl, err := loadConfigTemplate(*configTmpl)
	if err != nil {
		panic(err)
	}
	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	nebulaClient := clientset.NewForConfigOrDie(cfg)
	t, err := ParseTemplates(tmpls)
	if err != nil {
		log.Fatal(err)
	}
	mgr := Manager{
		kubeClient:   kubeClient,
		nebulaClient: nebulaClient,
		namespace:    *namespace,
		caName:       *caName,
		cfgTmpl:      cfgTmpl,
	}
	handler := Handler{
		mgr:   mgr,
		tmpls: t,
	}
	r := mux.NewRouter()
	r.HandleFunc("/api/sign", handler.sign)
	r.HandleFunc("/api/join", handler.join)
	r.HandleFunc("/api/approve", handler.approve)
	r.HandleFunc("/api/get/{name:[a-zA-z0-9-]+}", handler.get)
	r.HandleFunc("/node/{namespace:[a-zA-z0-9-]+}/{name:[a-zA-z0-9-]+}", handler.handleNode)
	r.HandleFunc("/ca/{namespace:[a-zA-z0-9-]+}/{name:[a-zA-z0-9-]+}", handler.handleCA)
	r.HandleFunc("/", handler.handleIndex)
	http.Handle("/", r)
	fmt.Printf("Starting HTTP server on port: %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
