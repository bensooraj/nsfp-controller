package main

import (
	"log"

	informersCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// NSFPController is an implementation of the Not Safe For Production dummy
// kubernetes controller that I am going to build for the sake of practise!
type NSFPController struct {
	PodGetter         typedCoreV1.PodsGetter
	PodLister         listerCoreV1.PodLister
	PodInformerSynced cache.InformerSynced
}

// NewNSFPController returns a new instance of the controller we are building!
func NewNSFPController(k8sClientset *kubernetes.Clientset, podInformer informersCoreV1.PodInformer) *NSFPController {

	nsfpController := &NSFPController{
		PodGetter:         k8sClientset.CoreV1(),
		PodLister:         podInformer.Lister(),
		PodInformerSynced: podInformer.Informer().HasSynced,
	}

	// Configure event handlers for pod events
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		// AddFunc: func(obj interface{}),
		// UpdateFunc: func(oldObj, newObj interface{}),
		// DeleteFunc: func(obj interface{}),
	})

	return nsfpController
}

// Run starts the controller!
func (nsfpc *NSFPController) Run(stopChannel <-chan struct{}) {
	if !cache.WaitForCacheSync(stopChannel, nsfpc.PodInformerSynced) {
		log.Println("Timeout while waiting for cache to populate!")
		return
	}
	log.Println("Caches synced...")

	// Wait until the stop signal is received
	log.Println("Waiting for stop signal.")
	<-stopChannel
	log.Println("Received stop signal.")
}
