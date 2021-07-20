package main

import (
	"context"
	"log"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/konveyor/crane-lib/state_transfer"
	"github.com/konveyor/crane-lib/state_transfer/endpoint"
	"github.com/konveyor/crane-lib/state_transfer/endpoint/route"
	"github.com/konveyor/crane-lib/state_transfer/labels"
	"github.com/konveyor/crane-lib/state_transfer/transfer"
	"github.com/konveyor/crane-lib/state_transfer/transfer/rclone"
	"github.com/konveyor/crane-lib/state_transfer/transport"
	"github.com/konveyor/crane-lib/state_transfer/transport/stunnel"
	routev1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	srcCfg       = &rest.Config{}
	destCfg      = &rest.Config{}
	srcNamespace = "robot-shop"
	srcPVC       = "mysql-data-volume-claim"
)

// This example shows how to wire up the components of the lib to
// transfer data from one PVC to another
func main() {
	destCfg, err := config.GetConfig()
	destCfg.Burst = 1000
	destCfg.QPS = 1000
	oldKubeConfigEnv := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", os.Getenv("HOME")+"/.kube/config")
	srcCfg, err := config.GetConfig()
	srcCfg.Burst = 1000
	srcCfg.QPS = 1000
	os.Setenv("KUBECONFIG", oldKubeConfigEnv)

	scheme := runtime.NewScheme()
	if err := routev1.AddToScheme(scheme); err != nil {
		log.Fatal(err, "unable to add routev1 scheme")
	}
	if err := v1.AddToScheme(scheme); err != nil {
		log.Fatal(err, "unable to add v1 scheme")
	}
	if err := corev1.AddToScheme(scheme); err != nil {

		log.Fatal(err, "unable to add corev1 scheme")
	}

	srcClient, err := client.New(srcCfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err, "unable to create source client")
	}

	destClient, err := client.New(destCfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err, "unable to create destination client")
	}

	// quiesce the applications if needed on the source side
	err = state_transfer.QuiesceApplications(srcCfg, srcNamespace)
	if err != nil {
		log.Fatal(err, "unable to quiesce application on source cluster")
	}

	// set up the PVC on destination to receive the data
	pvc := &corev1.PersistentVolumeClaim{}
	err = srcClient.Get(context.TODO(), client.ObjectKey{Namespace: srcNamespace, Name: srcPVC}, pvc)
	if err != nil {
		log.Fatal(err, "unable to get source PVC")
	}

	destPVC := pvc.DeepCopy()

	destPVC.ResourceVersion = ""
	destPVC.Spec.VolumeName = ""
	pvc.Annotations = map[string]string{}
	err = destClient.Create(context.TODO(), destPVC, &client.CreateOptions{})
	if err != nil {
		log.Fatal(err, "unable to create destination PVC")
	}

	pvcList, err := transfer.NewPVCPairList(
		transfer.NewPVCPair(pvc, destPVC),
	)
	if err != nil {
		log.Fatal(err, "invalid pvc list")
	}

	endpointPort := int32(2222)
	// create a route for data transfer
	r := route.NewEndpoint(
		types.NamespacedName{
			Namespace: pvc.Namespace,
			Name:      pvc.Name,
		}, endpointPort, route.EndpointTypePassthrough, labels.Labels)
	e, err := endpoint.Create(r, destClient)
	if err != nil {
		log.Fatal(err, "unable to create route endpoint")
	}

	_ = wait.PollUntil(time.Second*5, func() (done bool, err error) {
		ready, err := e.IsHealthy(destClient)
		if err != nil {
			log.Println(err, "unable to check route health, retrying...")
			return false, nil
		}
		return ready, nil
	}, make(<-chan struct{}))

	// create an stunnel transport to carry the data over the route
	proxyOptions := transport.ProxyOptions{
		URL:      "127.0.0.1",
		Username: "foo",
		Password: "bar",
	}
	s := stunnel.NewTransport(&proxyOptions)
	_, err = transport.CreateServer(s, srcClient, e)
	if err != nil {
		log.Fatal(err, "error creating stunnel client")
	}

	//_, err = transport.CreateClient(s, destClient, e)
	//if err != nil {
	//	log.Fatal(err, "error creating stunnel server")
	//}

	// Create Rclone Transfer Pod
	t, err := rclone.NewTransfer(s, r, srcCfg, destCfg, pvcList)
	if err != nil {
		log.Fatal(err, "errror creating rclone transfer")
	}

	err = transfer.CreateServer(t)
	if err != nil {
		log.Fatal(err, "error creating rclone server")
	}

	// Create Rclone Client Pod
	err = transfer.CreateClient(t)
	if err != nil {
		log.Fatal(err, "error creating rclone client")
	}

	// TODO: check if the client is completed
}
