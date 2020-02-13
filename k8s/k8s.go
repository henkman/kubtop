package k8s

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	IP            string
}

var (
	reTopNode  = regexp.MustCompile(`(?m)^(\S+)\s*([0-9]+)m\s*([0-9]+)%\s*([0-9]+)Mi\s*([0-9]+)%`)
	reTopPod   = regexp.MustCompile(`(?m)^(\S+)\s*(\S+)\s*([0-9]+)m\s*([0-9]+)Mi`)
	reMemoryMi = regexp.MustCompile(`([0-9]+)Mi`)
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

func GetTopPods(kubeconfig string, buffer *bytes.Buffer) ([]TopPod, error) {
	cmd := exec.Command("kubectl", "top", "pods", "-A", "--kubeconfig", kubeconfig)
	cmd.Stdout = buffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	m := reTopPod.FindAllStringSubmatch(buffer.String(), -1)
	if m == nil {
		return nil, errors.New("top pods did not contain any pods")
	}
	nodes := make([]TopPod, len(m))
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
	return nodes, nil
}

func GetPodDetails(kubeconfig string, buffer *bytes.Buffer) ([]PodDetails, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-A", "-o", "json", "--kubeconfig", kubeconfig)
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
				PodIP     string    `json:"podIP"`
				StartTime time.Time `json:"startTime"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(buffer).Decode(&raw); err != nil {
		return nil, err
	}
	fmt.Println(raw)
	pods := make([]PodDetails, len(raw.Items))
	for i, item := range raw.Items {
		memoryLimitMi := -1
		m := reMemoryMi.FindStringSubmatch(item.Spec.Containers[0].Resources.Limits.Memory)
		if m != nil {
			mm, err := strconv.Atoi(m[1])
			if err != nil {
				memoryLimitMi = mm
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
			IP:            item.Status.PodIP,
		}
	}
	return pods, nil

}
