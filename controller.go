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

// NSFPController is an implementation of the Not Safe For Production dummy
// kubernetes controller that I am going to build for the sake of practise!
type NSFPController struct {
	SecretGetter         typedCoreV1.SecretsGetter
	SecretLister         listerCoreV1.SecretLister
	SecretInformerSynced cache.InformerSynced
}

// NewNSFPController returns a new instance of the controller we are building!
func NewNSFPController(k8sClientset *kubernetes.Clientset, secretInformer informersCoreV1.SecretInformer) *NSFPController {

	nsfpController := &NSFPController{
		SecretGetter:         k8sClientset.CoreV1(),
		SecretLister:         secretInformer.Lister(),
		SecretInformerSynced: secretInformer.Informer().HasSynced,
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
	if !cache.WaitForCacheSync(stopChannel, nsfpc.SecretInformerSynced) {
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
}
