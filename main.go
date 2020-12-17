package main

import (
	"flag"
	"log"
	"os"
	"time"

	version "github.com/bensooraj/nsfp-controller/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	log.Printf("nsfp-controller version: %s", version.VERSION)

	kubeconfig := flag.String("kubeconfig", "", "kubeconfig file")
	flag.Parse()

	if *kubeconfig == "" {
		*kubeconfig = os.Getenv("KUBECONFIG")
	}

	var restConfig *rest.Config
	var err error

	if *kubeconfig != "" {
		log.Println("kubeconfig file found: ", *kubeconfig)
		restConfig, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		log.Println("kubeconfig file not found. Fetching in-cluster config")
		restConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalf("Error fetching the rest config: %v\n", err)
		os.Exit(1)
	}

	k8sClientset := kubernetes.NewForConfigOrDie(restConfig)

	// Informer
	sharedInformers := informers.NewSharedInformerFactory(k8sClientset, 30*time.Second)

	sharedInformers.Start(wait.NeverStop)
	sharedInformers.WaitForCacheSync(wait.NeverStop)
}
