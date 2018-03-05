package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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

// PatchNodeLabels patches node labels.
func PatchNodeLabels(c v1core.CoreV1Interface, nodeName types.NodeName, oldNode *v1.Node, newNode *v1.Node) (*v1.Node, []byte, error) {
	patchBytes, err := preparePatchBytesforNodeStatus(nodeName, oldNode, newNode)
	if err != nil {
		return nil, nil, err
	}

	updatedNode, err := c.Nodes().Patch(string(nodeName), types.StrategicMergePatchType, patchBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to patch objectmeta %q for node %q: %v", patchBytes, nodeName, err)
	}
	return updatedNode, patchBytes, nil
}

func preparePatchBytesforNodeStatus(nodeName types.NodeName, oldNode *v1.Node, newNode *v1.Node) ([]byte, error) {
	oldData, err := json.Marshal(oldNode)
	if err != nil {
		return nil, fmt.Errorf("failed to Marshal oldData for node %q: %v", nodeName, err)
	}

	newNode.Spec = oldNode.Spec
	newData, err := json.Marshal(newNode)
	if err != nil {
		return nil, fmt.Errorf("failed to Marshal newData for node %q: %v", nodeName, err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1.Node{})
	if err != nil {
		return nil, fmt.Errorf("failed to CreateTwoWayMergePatch for node %q: %v", nodeName, err)
	}
	return patchBytes, nil
}

func (c *NodeController) createNode(obj interface{}) {
	node := obj.(*v1.Node)
	needupdate := false
	// always when new node is created check that correct labels exist
	if val, ok := node.ObjectMeta.Labels[kubeletapis.LabelZoneFailureDomain]; ok {
		if val != "nova" {
			// correct zone
			needupdate = true
		}
	} else {
		// add zone nova
		needupdate = true
	}
	if needupdate {
		// correct zone
		newnode := node.DeepCopy()
		newnode.ObjectMeta.Labels[kubeletapis.LabelZoneFailureDomain] = "nova"
		result, _, err := PatchNodeLabels(c.kclient.CoreV1(), types.NodeName(node.Name), node, newnode)
		if err != nil {
			log.Println(fmt.Sprintf("Failed to update node: %s", err.Error()))
		} else {
			log.Println(fmt.Sprintf("Updated node %s label: %s = nova", result.Name, kubeletapis.LabelZoneFailureDomain))
		}
	}
}
