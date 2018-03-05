package controller

import (
	"log"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	restclient "k8s.io/client-go/rest"
	//v1 "github.com/openshift/api/build/v1"
	buildv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
)

// NodeController watches the kubernetes api for changes to nodes and
// modifies zone label in that particular node.
type NodeController struct {
	nodeInformer      cache.SharedIndexInformer
	kclient           *kubernetes.Clientset
}

// Run starts the process for listening for node changes and acting upon those changes.
func (c *NodeController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	// Execute go function
	go c.nodeInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

// NewNodeController creates a new NewNodeController
func NewNodeController(kclient *kubernetes.Clientset, config *restclient.Config) *NodeController {
	nodeWatcher := &NodeController{}

	buildV1Client, err := buildv1.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	buildV1Client.Builds(v1.NamespaceAll).List(metav1.ListOptions{})

	nodeInformer := cache.NewSharedIndexInformer(

		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kclient.Core().Nodes().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kclient.Core().Nodes().Watch(options)
			},
		},
		&v1.Node{},
		3*time.Minute,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: nodeWatcher.createNode,
	})

	nodeWatcher.kclient = kclient
	nodeWatcher.nodeInformer = nodeInformer

	return nodeWatcher
}


func (c *NodeController) createNode(obj interface{}) {
	node := obj.(*v1.Node)
	log.Println("foobar")
}
