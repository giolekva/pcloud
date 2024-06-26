package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	nebulav1 "github.com/giolekva/pcloud/core/nebula/controller/apis/nebula/v1"
	clientset "github.com/giolekva/pcloud/core/nebula/controller/generated/clientset/versioned"
	informers "github.com/giolekva/pcloud/core/nebula/controller/generated/informers/externalversions/nebula/v1"
	listers "github.com/giolekva/pcloud/core/nebula/controller/generated/listers/nebula/v1"
)

var secretImmutable = true

type caRef struct {
	key string
}

type nodeRef struct {
	key string
}

type NebulaController struct {
	kubeClient   kubernetes.Interface
	nebulaClient clientset.Interface
	caLister     listers.NebulaCALister
	caSynced     cache.InformerSynced
	nodeLister   listers.NebulaNodeLister
	nodeSynced   cache.InformerSynced
	secretLister corev1listers.SecretLister
	secretSynced cache.InformerSynced
	workqueue    workqueue.RateLimitingInterface
}

func NewNebulaController(kubeClient kubernetes.Interface,
	nebulaClient clientset.Interface,
	caInformer informers.NebulaCAInformer,
	nodeInformer informers.NebulaNodeInformer,
	secretInformer corev1informers.SecretInformer) *NebulaController {
	c := &NebulaController{
		kubeClient:   kubeClient,
		nebulaClient: nebulaClient,
		caLister:     caInformer.Lister(),
		caSynced:     caInformer.Informer().HasSynced,
		nodeLister:   nodeInformer.Lister(),
		nodeSynced:   nodeInformer.Informer().HasSynced,
		secretLister: secretInformer.Lister(),
		secretSynced: secretInformer.Informer().HasSynced,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nebula"),
	}

	caInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueCA,
		UpdateFunc: func(_, o interface{}) {
			c.enqueueCA(o)
		},
		DeleteFunc: func(o interface{}) {
		},
	})
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueNode,
		UpdateFunc: func(_, o interface{}) {
			c.enqueueNode(o)
		},
		DeleteFunc: func(o interface{}) {
		},
	})

	return c
}

func (c *NebulaController) enqueueCA(o interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(o); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(caRef{key})
}

func (c *NebulaController) enqueueNode(o interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(o); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(nodeRef{key})
}

func (c *NebulaController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	klog.Info("Starting NebulaCA controller")
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.caSynced, c.nodeSynced, c.secretSynced); !ok {
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

func (c *NebulaController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *NebulaController) processNextWorkItem() bool {
	o, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(o interface{}) error {
		defer c.workqueue.Done(o)
		if ref, ok := o.(caRef); ok {
			if err := c.processCAWithKey(ref.key); err != nil {
				c.workqueue.AddRateLimited(ref)
				return fmt.Errorf("Error syncing '%s': %s, requeuing", ref.key, err.Error())
			}
			c.workqueue.Forget(o)
			fmt.Printf("Successfully synced CA '%s'\n", ref.key)
		} else if ref, ok := o.(nodeRef); ok {
			if err := c.processNodeWithKey(ref.key); err != nil {
				c.workqueue.AddRateLimited(ref)
				return fmt.Errorf("Error syncing '%s': %s, requeuing", ref.key, err.Error())
			}
			c.workqueue.Forget(o)
			fmt.Printf("Successfully synced Node '%s'\n", ref.key)
		} else {
			c.workqueue.Forget(o)
			utilruntime.HandleError(fmt.Errorf("expected reference in workqueue but got %#v", o))
			return nil
		}
		c.workqueue.Forget(o)
		return nil
	}(o)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *NebulaController) processCAWithKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil
	}
	ca, err := c.caLister.NebulaCAs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("CA '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	if ca.Status.State == nebulav1.NebulaCAStateReady {
		fmt.Printf("%s CA is already in Ready state\n", ca.Name)
		return nil
	}
	privKey, cert, err := CreateCertificateAuthority(apiAddr(ca.Name), ca.Name)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: ca.Spec.SecretName,
		},
		Immutable: &secretImmutable,
		Data: map[string][]byte{
			"ca.key": privKey,
			"ca.crt": cert,
		},
	}
	_, err = c.kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	err = c.updateCAStatus(ca, nebulav1.NebulaCAStateReady, "Generated credentials")
	if err != nil {
		return err
	}
	return nil
}

func (c *NebulaController) processNodeWithKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil
	}
	node, err := c.nodeLister.NebulaNodes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("NebulaNode '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	if node.Status.State == nebulav1.NebulaNodeStateReady {
		fmt.Printf("%s Node is already in Ready state\n", node.Name)
		return nil
	}
	ca, err := c.caLister.NebulaCAs(node.Spec.CANamespace).Get(node.Spec.CAName)
	if err != nil {
		return err
	}
	if ca.Status.State != nebulav1.NebulaCAStateReady {
		return fmt.Errorf("Referenced CA %s is not ready yet.", node.Spec.CAName)
	}
	caSecret, err := c.secretLister.Secrets(ca.Namespace).Get(ca.Spec.SecretName)
	if err != nil {
		if errors.IsNotFound(err) {
			c.updateNodeStatus(node, nebulav1.NebulaNodeStateError, "Could not find CA secret")
		}
		return err
	}
	var pubKey []byte
	if node.Spec.PubKey != "" {
		pubKey = []byte(node.Spec.PubKey)
	}
	privKey, nodeCert, err := SignNebulaNode(apiAddr(ca.Name), caSecret.Data["ca.key"], caSecret.Data["ca.crt"], node.Name, pubKey, node.Spec.IPCidr)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: node.Spec.SecretName,
		},
		Immutable: &secretImmutable,
		Data: map[string][]byte{
			"ca.crt":   caSecret.Data["ca.crt"],
			"host.crt": nodeCert,
			"host.key": privKey,
		},
	}
	_, err = c.kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	err = c.updateNodeStatus(node, nebulav1.NebulaNodeStateReady, "Generated credentials")
	if err != nil {
		return err
	}
	return nil
}

func (c *NebulaController) updateCAStatus(ca *nebulav1.NebulaCA, state nebulav1.NebulaCAState, msg string) error {
	cp := ca.DeepCopy()
	cp.Status.State = state
	cp.Status.Message = msg
	_, err := c.nebulaClient.LekvaV1().NebulaCAs(cp.Namespace).UpdateStatus(context.TODO(), cp, metav1.UpdateOptions{})
	return err
}

func (c *NebulaController) updateNodeStatus(node *nebulav1.NebulaNode, state nebulav1.NebulaNodeState, msg string) error {
	cp := node.DeepCopy()
	cp.Status.State = state
	cp.Status.Message = msg
	_, err := c.nebulaClient.LekvaV1().NebulaNodes(cp.Namespace).UpdateStatus(context.TODO(), cp, metav1.UpdateOptions{})
	return err
}

// TODO(giolekva): maybe pass by flag?
func apiAddr(ca string) string {
	return fmt.Sprintf("http://nebula-api.%s-ingress-private.svc.cluster.local", ca)
}
