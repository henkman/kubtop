package k8s

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

type TopNode struct {
	Name          string
	MilliCPU      int
	CPUPercent    int
	MemoryMi      int
	MemoryPercent int
}

type TopPod struct {
	Namespace string
	Name      string
	MilliCPU  int
	MemoryMi  int
}

type PodDetails struct {
	Name          string
	Namespace     string
	NodeName      string
	Image         string
	MemoryLimitMi int
	Phase         string
	PhaseStart    time.Time
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

var (
	reTopNode   = regexp.MustCompile(`(?m)^(\S+)\s*([0-9]+)m\s*([0-9]+)%\s*([0-9]+)Mi\s*([0-9]+)%`)
	reTopPodAll = regexp.MustCompile(`(?m)^(\S+)\s*(\S+)\s*([0-9]+)m\s*([0-9]+)Mi`)
	reTopPod    = regexp.MustCompile(`(?m)^(\S+)\s*([0-9]+)m\s*([0-9]+)Mi`)
	reMemoryMi  = regexp.MustCompile(`([0-9]+)(Mi|Gi)`)
)

func GetTopNode(kubeconfig string, buffer *bytes.Buffer) ([]TopNode, error) {
	cmd := exec.Command("kubectl", "top", "node", "--kubeconfig", kubeconfig)
	cmd.Stdout = buffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	m := reTopNode.FindAllStringSubmatch(buffer.String(), -1)
	if m == nil {
		return nil, errors.New("top node did not contain any nodes")
	}
	nodes := make([]TopNode, len(m))
	for i, sm := range m {
		milliCPU, err := strconv.Atoi(sm[2])
		if err != nil {
			return nil, err
		}
		cPUPercent, err := strconv.Atoi(sm[3])
		if err != nil {
			return nil, err
		}
		memory, err := strconv.Atoi(sm[4])
		if err != nil {
			return nil, err
		}
		memoryPercent, err := strconv.Atoi(sm[5])
		if err != nil {
			return nil, err
		}
		nodes[i] = TopNode{
			Name:          sm[1],
			MilliCPU:      milliCPU,
			CPUPercent:    cPUPercent,
			MemoryMi:      memory,
			MemoryPercent: memoryPercent,
		}
	}
	return nodes, nil
}

func GetTopPods(kubeconfig string, buffer *bytes.Buffer, allNamespaces bool) ([]TopPod, error) {
	var cmd *exec.Cmd
	if allNamespaces {
		cmd = exec.Command("kubectl", "top", "pods", "-A", "--kubeconfig", kubeconfig)
	} else {
		cmd = exec.Command("kubectl", "top", "pods", "--kubeconfig", kubeconfig)
	}
	cmd.Stdout = buffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var m [][]string
	if allNamespaces {
		m = reTopPodAll.FindAllStringSubmatch(buffer.String(), -1)
	} else {
		m = reTopPod.FindAllStringSubmatch(buffer.String(), -1)
	}
	if m == nil {
		return nil, errors.New("top pods did not contain any pods")
	}
	nodes := make([]TopPod, len(m))
	if allNamespaces {
		for i, sm := range m {
			milliCPU, err := strconv.Atoi(sm[3])
			if err != nil {
				return nil, err
			}
			memoryMi, err := strconv.Atoi(sm[4])
			if err != nil {
				return nil, err
			}
			nodes[i] = TopPod{
				Namespace: sm[1],
				Name:      sm[2],
				MilliCPU:  milliCPU,
				MemoryMi:  memoryMi,
			}
		}
	} else {
		for i, sm := range m {
			milliCPU, err := strconv.Atoi(sm[2])
			if err != nil {
				return nil, err
			}
			memoryMi, err := strconv.Atoi(sm[3])
			if err != nil {
				return nil, err
			}
			nodes[i] = TopPod{
				Namespace: "",
				Name:      sm[1],
				MilliCPU:  milliCPU,
				MemoryMi:  memoryMi,
			}
		}
	}
	return nodes, nil
}

func GetPodDetails(kubeconfig string, buffer *bytes.Buffer, allNamespaces bool) ([]PodDetails, error) {
	var cmd *exec.Cmd
	if allNamespaces {
		cmd = exec.Command("kubectl", "get", "pods", "-A", "-o", "json", "--kubeconfig", kubeconfig)
	} else {
		cmd = exec.Command("kubectl", "get", "pods", "-o", "json", "--kubeconfig", kubeconfig)
	}
	cmd.Stdout = buffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var raw struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Spec struct {
				Containers []struct {
					Image     string `json:"image"`
					Resources struct {
						Limits struct {
							Memory string `json:"memory"`
						} `json:"limits"`
					} `json:"resources"`
				} `json:"containers"`
				NodeName string `json:"nodeName"`
			} `json:"spec"`
			Status struct {
				Phase     string    `json:"phase"`
				StartTime time.Time `json:"startTime"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(buffer).Decode(&raw); err != nil {
		return nil, err
	}
	pods := make([]PodDetails, len(raw.Items))
	for i, item := range raw.Items {
		memoryLimitMi := -1
		m := reMemoryMi.FindStringSubmatch(item.Spec.Containers[0].Resources.Limits.Memory)
		if m != nil {
			mm, err := strconv.Atoi(m[1])
			if err == nil {
				if m[2] == "Gi" {
					memoryLimitMi = mm * 1024
				} else {
					memoryLimitMi = mm
				}
			}
		}
		pods[i] = PodDetails{
			Name:          item.Metadata.Name,
			Namespace:     item.Metadata.Namespace,
			NodeName:      item.Spec.NodeName,
			Image:         item.Spec.Containers[0].Image,
			MemoryLimitMi: memoryLimitMi,
			Phase:         item.Status.Phase,
			PhaseStart:    item.Status.StartTime,
		}
	}
	return pods, nil

}

func GetNodeOverview(kubeconfig string, buffer *bytes.Buffer, allNamespaces bool) ([]Node, error) {
	topnodes, err := GetTopNode(kubeconfig, buffer)
	if err != nil {
		return nil, err
	}
	buffer.Reset()
	toppods, err := GetTopPods(kubeconfig, buffer, allNamespaces)
	if err != nil {
		return nil, err
	}
	buffer.Reset()
	poddetails, err := GetPodDetails(kubeconfig, buffer, allNamespaces)
	if err != nil {
		return nil, err
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
	getTopPodByName := func(name string) (TopPod, error) {
		for _, tp := range toppods {
			if tp.Name == name {
				return tp, nil
			}
		}
		return TopPod{}, errors.New("toppod " + name + " not found")
	}

	for _, pd := range poddetails {
		pod := Pod{
			Name:          pd.Name,
			MemoryLimitMi: pd.MemoryLimitMi,
			Image:         pd.Image,
			Phase:         pd.Phase,
			PhaseStart:    pd.PhaseStart,
		}
		tp, err := getTopPodByName(pd.Name)
		if err == nil {
			pod.MilliCPU = tp.MilliCPU
			pod.MemoryMi = tp.MemoryMi
		}
		addPodToNode(pod, pd.NodeName)
	}

	return nodes, nil
}
