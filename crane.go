package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jmontleon/crane-lib/state_transfer"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const ns = "robot-shop"

func main() {

	dstcfg, err := config.GetConfig()
	dstcfg.Burst = 1000
	dstcfg.QPS = 1000
	oldKubeConfigEnv := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", os.Getenv("HOME")+"/.kube-src/config")
	srccfg, err := config.GetConfig()
	srccfg.Burst = 1000
	srccfg.QPS = 1000
	os.Setenv("KUBECONFIG", oldKubeConfigEnv)

	scheme := runtime.NewScheme()
	if err := projectv1.AddToScheme(scheme); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c, err := client.New(srccfg, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	project := projectv1.Project{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: ns, Namespace: ns}, &project)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	project.ResourceVersion = ""

	dc, err := client.New(dstcfg, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = dc.Create(context.TODO(), &project, &client.CreateOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = state_transfer.QuiesceApplications(srccfg, ns)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	nsPVCList := v1.PersistentVolumeClaimList{}

	err = c.List(context.TODO(), &nsPVCList, &client.ListOptions{Namespace: ns})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, pvc := range nsPVCList.Items {
		pvc.ResourceVersion = ""
		pvc.Spec.VolumeName = ""
		pvc.Annotations = map[string]string{}
		err = dc.Create(context.TODO(), &pvc, &client.CreateOptions{})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	for _, pvc := range nsPVCList.Items {
		var transfer state_transfer.transfer
		transfer = state_transfer.CreateRsynctransfer()
		transfer.SetTransport(&state_transfer.StunnelTransport{})
		//transfer.SetEndpoint(&state_transfer.RouteEndpoint{})
		transfer.SetEndpoint(&state_transfer.LoadBalancerEndpoint{})
		transfer.SetSource(srccfg)
		transfer.SetDestination(dstcfg)
		transfer.SetPVC(pvc)
		err := state_transfer.CreateServer(transfer)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = state_transfer.CreateClient(transfer)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
