package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	version "github.com/bensooraj/nsfp-controller/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	log.Printf("nsfp-controller version: %s", version.VERSION)

	ctx := context.Background()

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

	// List all the nodes!
	nodeList, err := k8sClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing the nodes: %v\n", err)
		os.Exit(2)
	}
	for index, node := range nodeList.Items {
		log.Printf("Node #%d: %s | %+v\n", index, node.Name, node.Labels)
	}

	// Informer
	sharedInformers := informers.NewSharedInformerFactory(k8sClientset, 30*time.Second)
	nsfpController := NewNSFPController(k8sClientset, sharedInformers.Core().V1().Pods())

	stopChannel := make(chan struct{}, 0)
	defer close(stopChannel)

	sharedInformers.Start(wait.NeverStop)
	// sharedInformers.WaitForCacheSync(wait.NeverStop)
	nsfpController.Run(stopChannel)

}
