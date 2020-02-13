package k8s_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/henkman/kubtop/k8s"
)

const KUBECONFIG = "../dev.conf"

func TestGetTopNode(t *testing.T) {
	return
	var buf bytes.Buffer
	fmt.Println(k8s.GetTopNode(KUBECONFIG, &buf))
}

func TestGetTopPods(t *testing.T) {
	return
	var buf bytes.Buffer
	fmt.Println(k8s.GetTopPods(KUBECONFIG, &buf))
}

func TestGetPodDetails(t *testing.T) {
	return
	var buf bytes.Buffer
	fmt.Println(k8s.GetPodDetails(KUBECONFIG, &buf))
}

func TestNodeOverview(t *testing.T) {
	var buf bytes.Buffer
	topnodes, err := k8s.GetTopNode(KUBECONFIG, &buf)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	buf.Reset()
	toppods, err := k8s.GetTopPods(KUBECONFIG, &buf)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	buf.Reset()
	poddetails, err := k8s.GetPodDetails(KUBECONFIG, &buf)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	type Pod struct {
		Name          string
		MilliCPU      int
		MemoryMi      int
		MemoryLimitMi int
		Image         string
		Phase         string
		PhaseStart    time.Time
		IP            string
	}
	type Node struct {
		Name          string
		MilliCPU      int
		CPUPercent    int
		MemoryMi      int
		MemoryPercent int
		Pods          []Pod
	}
	nodes := make([]Node, len(topnodes))
	for i, tn := range topnodes {
		nodes[i] = Node{
			Name:          tn.Name,
			MilliCPU:      tn.MilliCPU,
			CPUPercent:    tn.CPUPercent,
			MemoryMi:      tn.MemoryMi,
			MemoryPercent: tn.MemoryPercent,
			Pods:          []Pod{},
		}
	}

	addPodToNode := func(p Pod, nodeName string) {
		for i, node := range nodes {
			if node.Name == nodeName {
				nodes[i].Pods = append(nodes[i].Pods, p)
				break
			}
		}
	}

	getTopPodByName := func(name string) (k8s.TopPod, error) {
		for _, tp := range toppods {
			if tp.Name == name {
				return tp, nil
			}
		}
		return k8s.TopPod{}, errors.New("toppod " + name + " not found")
	}

	for _, pd := range poddetails {
		tp, err := getTopPodByName(pd.Name)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}

		pod := Pod{
			Name:          pd.Name,
			MilliCPU:      tp.MilliCPU,
			MemoryMi:      tp.MemoryMi,
			MemoryLimitMi: pd.MemoryLimitMi,
			Image:         pd.Image,
			Phase:         pd.Phase,
			PhaseStart:    pd.PhaseStart,
			IP:            pd.IP,
		}
		addPodToNode(pod, pd.NodeName)
	}

	for _, node := range nodes {
		fmt.Printf("Name='%s' MilliCPU='%d' CPUPercent='%d' MemoryMi='%d' MemoryPercent='%d'\n",
			node.Name, node.MilliCPU, node.CPUPercent, node.MemoryMi, node.MemoryPercent)
		fmt.Println("\tName, MilliCPU, MemoryMi, MemoryLimitMi, Image, Phase, PhaseStart, IP")
		for _, pod := range node.Pods {
			fmt.Println("\t"+pod.Name, pod.MilliCPU, pod.MemoryMi, pod.MemoryLimitMi, pod.Image, pod.Phase, time.Since(pod.PhaseStart), pod.PhaseStart)
		}
	}
}
