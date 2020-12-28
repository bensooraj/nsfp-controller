package main

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	informersCoreV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SecretSyncType is the type of secrets we are watching...
	SecretSyncType = "bensooraj.com/secretsync"
	// SecretSyncAnnotation ...
	SecretSyncAnnotation = "bensooraj.com/secretsync"
	// SecretSyncSourceNamespace is the namespace we are watching
	SecretSyncSourceNamespace = "secretsync"
)

var namepaceBlacklist = map[string]bool{
	"kube-node-lease":         true,
	"kube-public":             true,
	"kube-system":             true,
	"local-path-storage":      true,
	SecretSyncSourceNamespace: true,
}

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

	nsfpc.doSync()

	// Wait until the stop signal is received
	log.Println("Waiting for stop signal.")
	// <-stopChannel
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

	// nsfpc.HandleSecretChange(obj)
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

	// nsfpc.HandleSecretChange(obj)
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

	// List the namespaces
	nss, err := nsfpc.NamespaceLister.List(labels.Everything())
	if err != nil {
		log.Printf("Error listing namespaces: %v", err)
		return
	}

	for _, ns := range nss {
		nsName := ns.ObjectMeta.Name

		if _, ok := namepaceBlacklist[nsName]; ok {
			log.Printf("Skipping the namespace %s\n", nsName)
			continue
		}

		log.Printf("Copy the secret %s over to the namespace %s\n", secret.ObjectMeta.Name, ns.ObjectMeta.Name)

		// nsfpc.copySecretToNamespace(secret, nsName)
	}
}

func (nsfpc *NSFPController) doSync() {
	sourceSecrets, err := nsfpc.SecretLister.Secrets(SecretSyncSourceNamespace).List(labels.Everything())
	if err != nil {
		log.Panicf("Listing secrets... Oh noes!: %v\n", err)
		runtime.HandleError(err)
	}

	var filteredSecrets []*coreV1.Secret

	for _, secret := range sourceSecrets {
		if annotation, found := secret.Annotations["bensooraj.com/secretsync"]; found && annotation == "true" {
			log.Printf("RELEVANT\n")
			log.Printf("Syncing secret %s/%s\n", secret.Namespace, secret.Name)

			filteredSecrets = append(filteredSecrets, secret)
		}
	}
	log.Printf("filteredSecrets: %+v\n", filteredSecrets)

	rawNamepaces, err := nsfpc.NamespaceLister.List(labels.Everything())
	if err != nil {
		log.Panicf("Listing namespaces... Oh noes!: %v\n", err)
		runtime.HandleError(err)
	}

	var destinationNamepaces []*coreV1.Namespace
	for _, ns := range rawNamepaces {
		if annotation, found := ns.Annotations["bensooraj.com/secretsync"]; found && annotation == "true" {
			destinationNamepaces = append(destinationNamepaces, ns)
		}
	}

	for _, ns := range destinationNamepaces {
		log.Printf("Destination namespace %s\n", ns.Name)
		nsfpc.syncNamespace(filteredSecrets, ns.Name)
	}
}

func (nsfpc *NSFPController) syncNamespace(secrets []*coreV1.Secret, nsName string) {
	ctx := context.Background()

	// TODO:
	// 1. Create/Update all of the secrets in this namespace
	// 2. Delete secrets that have the annotation but are not in the filtered list of secrets

	for _, secret := range secrets {
		newSecret := secret.DeepCopy()

		newSecret.Namespace = nsName
		newSecret.ResourceVersion = "" // resourceVersion should not be set on objects to be created
		newSecret.UID = "" // Precondition failed: UID in precondition
		log.Printf("New secret copied over: %v\n", newSecret)

		// Create
		_, err := nsfpc.SecretGetter.Secrets(nsName).Create(ctx, newSecret, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.Printf("ALREADY EXISTS: %+v\n", err)
				// Update if the secret already exists
				_, updateErr := nsfpc.SecretGetter.Secrets(nsName).Update(ctx, newSecret, metav1.UpdateOptions{})
				if updateErr != nil {
					log.Printf("Error updating %s/%s: %+v\n", newSecret.Namespace, newSecret.Name, updateErr)
				}
			} else {
				log.Printf("Error creating the new secret: %+v\n", err)
			}
		}
	}

	// TODO:
	// 1. Make a deep copy of the secret
	// 2. Remove things like object versions that will prevent us from writing
	// 3. Write in the new namespace
	// 4. Create/Update for the new object
}
