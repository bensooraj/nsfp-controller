package main

import (
	"log"

	"k8s.io/apimachinery/pkg/util/runtime"
	informersCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	coreV1 "k8s.io/api/core/v1"
)

const (
	// SecretSyncType is the type of secrets we are watching...
	SecretSyncType = "bensooraj.com/secretSync"
	// SecretSyncSourceNamespace is the namespace we are watching
	SecretSyncSourceNamespace = "secretSync"
)

// NSFPController is an implementation of the Not Safe For Production dummy
// kubernetes controller that I am going to build for the sake of practise!
type NSFPController struct {
	// Secrets
	SecretGetter         typedCoreV1.SecretsGetter
	SecretLister         listerCoreV1.SecretLister
	SecretInformerSynced cache.InformerSynced

	// Namespaces
	NamespaceGetter         typedCoreV1.NamespacesGetter
	NamespaceLister         listerCoreV1.NamespaceLister
	NamespaceInformerSynced cache.InformerSynced
}

// NewNSFPController returns a new instance of the controller we are building!
func NewNSFPController(
	k8sClientset *kubernetes.Clientset,
	secretInformer informersCoreV1.SecretInformer,
	namespaceInformer informersCoreV1.NamespaceInformer,
) *NSFPController {

	nsfpController := &NSFPController{
		// Secrets
		SecretGetter:         k8sClientset.CoreV1(),
		SecretLister:         secretInformer.Lister(),
		SecretInformerSynced: secretInformer.Informer().HasSynced,
		// Namespaces
		NamespaceGetter:         k8sClientset.CoreV1(),
		NamespaceLister:         namespaceInformer.Lister(),
		NamespaceInformerSynced: namespaceInformer.Informer().HasSynced,
	}

	// Configure event handlers for secret events
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { nsfpController.OnAdd(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { nsfpController.OnUdpate(oldObj, newObj) },
		DeleteFunc: func(obj interface{}) { nsfpController.OnDelete(obj) },
	})

	return nsfpController
}

// Run starts the controller!
func (nsfpc *NSFPController) Run(stopChannel <-chan struct{}) {
	if !cache.WaitForCacheSync(stopChannel, nsfpc.SecretInformerSynced, nsfpc.NamespaceInformerSynced) {
		log.Println("Timeout while waiting for cache to populate!")
		return
	}
	log.Println("Caches synced...")

	// Wait until the stop signal is received
	log.Println("Waiting for stop signal.")
	<-stopChannel
	log.Println("Received stop signal.")
}

// OnAdd handles secret addition events
func (nsfpc *NSFPController) OnAdd(obj interface{}) {
	secret := obj.(*coreV1.Secret)

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("[ADD ERROR] Error getting key for %#v: %v\n", obj, err)
		runtime.HandleError(err)
	}
	log.Printf("[ADD] %v | %s in the namepace %s\n", key, secret.Name, secret.Namespace)

	nsfpc.HandleSecretChange(obj)
}

// OnUdpate handles secret updation events
func (nsfpc *NSFPController) OnUdpate(oldObj, newObj interface{}) {
	secret := oldObj.(*coreV1.Secret)

	key, err := cache.MetaNamespaceKeyFunc(oldObj)
	if err != nil {
		log.Printf("[UPDATE ERROR] Error getting key for %#v: %v\n", oldObj, err)
		runtime.HandleError(err)
	}
	log.Printf("[UPDATE] %v | %s in the namepace %s\n", key, secret.Name, secret.Namespace)

	nsfpc.HandleSecretChange(newObj)
}

// OnDelete handles secret deletion events
func (nsfpc *NSFPController) OnDelete(obj interface{}) {
	secret := obj.(*coreV1.Secret)

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("[DELETE ERROR] Error getting key for %#v: %v\n", obj, err)
		runtime.HandleError(err)
	}
	log.Printf("[DELETE] %v | %s in the namepace %s\n", key, secret.Name, secret.Namespace)

	nsfpc.HandleSecretChange(obj)
}

// HandleSecretChange handles changes to a secret of the type and in the namespace we are watching
func (nsfpc *NSFPController) HandleSecretChange(obj interface{}) {
	secret, ok := obj.(*coreV1.Secret)
	if !ok {
		// TODO::DeletedFinalStateUnknown
		return
	}

	if secret.ObjectMeta.Namespace != SecretSyncSourceNamespace {
		log.Printf("Skipping secret %s in the wrong namespace %s\n", secret.Name, secret.ObjectMeta.Namespace)
		return
	}

	if secret.Type != SecretSyncType {
		log.Printf("Skipping secret %s of the wrong type %s\n", secret.Name, secret.Type)
		return
	}
}
