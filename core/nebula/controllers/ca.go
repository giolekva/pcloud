package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	nebulav1 "github.com/giolekva/pcloud/core/nebula/apis/nebula/v1"
	clientset "github.com/giolekva/pcloud/core/nebula/generated/clientset/versioned"
	informers "github.com/giolekva/pcloud/core/nebula/generated/informers/externalversions"
	listers "github.com/giolekva/pcloud/core/nebula/generated/listers/nebula/v1"
)

var secretImmutable = true

type CAController struct {
	kubeClient   kubernetes.Interface
	nebulaClient clientset.Interface
	caLister     listers.NebulaCALister
	caSynced     cache.InformerSynced
	workqueue    workqueue.RateLimitingInterface

	nebulaCert string
}

func NewCAController(kubeClient kubernetes.Interface, nebulaClient clientset.Interface, nebulaInformerFactory informers.SharedInformerFactory, nebulaCert string) *CAController {
	nebulaInformer := nebulaInformerFactory.Lekva().V1().NebulaCAs().Informer()
	c := &CAController{
		kubeClient:   kubeClient,
		nebulaClient: nebulaClient,
		caLister:     nebulaInformerFactory.Lekva().V1().NebulaCAs().Lister(),
		caSynced:     nebulaInformer.HasSynced,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NebulaCAs"),
		nebulaCert:   nebulaCert,
	}

	nebulaInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, o interface{}) {
			c.enqueue(o)
		},
		DeleteFunc: func(o interface{}) {
		},
	})

	return c
}

func (c *CAController) enqueue(o interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(o); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *CAController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	klog.Info("Starting NebulaCA controller")
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.caSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}
	fmt.Println("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	fmt.Println("Started workers")
	<-stopCh
	fmt.Println("Shutting down workers")
	return nil
}

func (c *CAController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *CAController) processNextWorkItem() bool {
	o, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(o interface{}) error {
		defer c.workqueue.Done(o)
		var key string
		var ok bool
		if key, ok = o.(string); !ok {
			c.workqueue.Forget(o)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", o))
			return nil
		}
		if err := c.processCAWithKey(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("Rrror syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(o)
		fmt.Printf("Successfully synced '%s'\n", key)
		return nil
	}(o)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *CAController) processCAWithKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil
	}
	ca, err := c.getCA(namespace, name)
	if err != nil {
		panic(err)
	}
	if ca.Status.State == nebulav1.NebulaCAStateReady {
		fmt.Printf("%s CA is already in Ready state\n", ca.Name)
		return nil
	}
	keyDir, err := generateCAKey(ca.Spec.CAName, c.nebulaCert)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(keyDir)
	secret, err := createSecretFromDir(keyDir)
	if err != nil {
		panic(err)
	}
	secret.Immutable = &secretImmutable
	secret.Name = ca.Spec.SecretName
	_, err = c.kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	err = c.updateStatus(ca, nebulav1.NebulaCAStateReady, "Generated credentials")
	if err != nil {
		panic(err)
	}
	return nil
}

func (c *CAController) updateStatus(ca *nebulav1.NebulaCA, state nebulav1.NebulaCAState, msg string) error {
	cp := ca.DeepCopy()
	cp.Status.State = state
	cp.Status.Message = msg
	_, err := c.nebulaClient.LekvaV1().NebulaCAs(cp.Namespace).UpdateStatus(context.TODO(), cp, metav1.UpdateOptions{})
	return err
}

func createSecretFromDir(path string) (*corev1.Secret, error) {
	all, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		Data: make(map[string][]byte),
	}
	for _, f := range all {
		if f.IsDir() {
			continue
		}
		d, err := ioutil.ReadFile(filepath.Join(path, f.Name()))
		if err != nil {
			return nil, err
		}
		secret.Data[f.Name()] = d
	}
	return secret, nil
}

func generateCAKey(name, nebulaCert string) (string, error) {
	tmp, err := os.MkdirTemp("", name)
	if err != nil {
		return "", err
	}
	fmt.Println(tmp)
	cmd := exec.Command(nebulaCert, "ca",
		"-name", name,
		"-out-key", filepath.Join(tmp, "ca.key"),
		"-out-crt", filepath.Join(tmp, "ca.crt"),
		"-out-qr", filepath.Join(tmp, "ca.png"))
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return tmp, nil
}

func (c *CAController) getCA(namespace, name string) (*nebulav1.NebulaCA, error) {
	s := labels.NewSelector()
	r, err := labels.NewRequirement("metadata.namespace", selection.Equals, []string{namespace})
	if err != nil {
		panic(err)
	}
	r1, err := labels.NewRequirement("metadata.name", selection.Equals, []string{name})
	if err != nil {
		panic(err)
	}
	s.Add(*r, *r1)
	ncas, err := c.caLister.List(s)
	if err != nil {
		panic(err)
	}
	if len(ncas) != 1 {
		panic("err")
	}
	return ncas[0], nil
}
