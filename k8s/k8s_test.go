package k8s_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/henkman/kubtop/k8s"
)

const KUBECONFIG = "../dev.conf"

func TestNodeOverview(t *testing.T) {
	var buf bytes.Buffer
	fmt.Println(k8s.GetNodeOverview(KUBECONFIG, &buf, false))
}
