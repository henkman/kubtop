package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hako/durafmt"

	"github.com/henkman/kubtop/k8s"
	"github.com/jedib0t/go-pretty/table"
	tui "github.com/marcusolsson/tui-go"
)

type Stage struct {
	Name       string `json:"name"`
	ConfigFile string `json:"configFile"`
}

func main() {
	root := tui.NewHBox()

	var config struct {
		AllNamespaces  bool    `json:"allNamespaces"`
		RefreshSeconds int     `json:"refreshSeconds`
		StageIndex     int     `json:"stageIndex`
		Stages         []Stage `json:"stages`
	}
	{
		exe, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			return
		}
		dir := filepath.Dir(exe)
		fd, err := os.Open(filepath.Join(dir, "kubtop.json"))
		if err != nil {
			fmt.Println(err)
			return
		}
		if err := json.NewDecoder(fd).Decode(&config); err != nil {
			fd.Close()
			fmt.Println(err)
			return
		}
		fd.Close()
	}

	labels := []*tui.Label{}
	boxes := []*tui.Box{}

	refreshText := tui.NewLabel(fmt.Sprint(config.RefreshSeconds) + "s")
	stageText := tui.NewLabel(config.Stages[config.StageIndex].Name)
	allText := tui.NewLabel(map[bool]string{true: "all", false: "default"}[config.AllNamespaces])

	infoBox := tui.NewHBox(refreshText, stageText, allText)
	infoBox.SetBorder(true)

	v := tui.NewVBox()
	base := tui.NewVBox(infoBox, v)
	scrollbox := tui.NewScrollArea(base)
	root.Append(scrollbox)

	ui, err := tui.New(root)
	if err != nil {
		log.Fatal(err)
	}
	var refreshLock sync.Mutex
	var buf bytes.Buffer
	refresh := func() {
		refreshLock.Lock()
		defer refreshLock.Unlock()

		nodes, err := k8s.GetNodeOverview(config.Stages[config.StageIndex].ConfigFile, &buf, config.AllNamespaces)
		buf.Reset()
		if err != nil {
			for i := 0; i < v.Length(); i++ {
				v.Remove(v.Length() - 1)
			}
			return
		}

		if len(nodes) > len(boxes) {
			d := len(nodes) - len(boxes)
			for i := 0; i < d; i++ {
				label := tui.NewLabel("")
				box := tui.NewVBox(label)
				box.SetBorder(true)
				labels = append(labels, label)
				boxes = append(boxes, box)
				v.Append(box)
			}
		} else if len(nodes) < len(boxes) {
			d := len(boxes) - len(nodes)
			for i := 0; i < d; i++ {
				v.Remove(v.Length() - 1)
			}
			boxes = boxes[:len(nodes)]
			labels = labels[:len(nodes)]
		}

		pcs := make([]int, len(nodes))
		for i, node := range nodes {
			pcs[i] = len(node.Pods)
		}

		for i, node := range nodes {
			boxes[i].SetTitle(node.Name + fmt.Sprintf(" [CPU = %dm(%d%%) Mem = %dMi(%d%%)]",
				node.MilliCPU, node.CPUPercent, node.MemoryMi, node.MemoryPercent))

			t := table.NewWriter()
			t.AppendHeader(table.Row{"Name", "Phase", "Since", "CPU", "Memory",
				"Limit", "Image"})
			for _, pod := range node.Pods {
				since := durafmt.ParseShort(time.Since(pod.PhaseStart)).LimitFirstN(2)
				t.AppendRow(table.Row{
					pod.Name, pod.Phase, since, fmt.Sprint(pod.MilliCPU, "m"),
					fmt.Sprint(pod.MemoryMi, "Mi"), fmt.Sprint(pod.MemoryLimitMi, "Mi"), pod.Image,
				})
			}
			t.Style().Options.DrawBorder = false
			t.Style().Options.SeparateHeader = false
			t.Style().Options.SeparateColumns = false
			labels[i].SetText(t.Render())
		}
		allText.SetText(map[bool]string{true: "all", false: "default"}[config.AllNamespaces])
		stageText.SetText(config.Stages[config.StageIndex].Name)
		ui.Repaint()
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })
	ui.SetKeybinding("c", func() { ui.Quit() })
	ui.SetKeybinding("q", func() { ui.Quit() })
	ui.SetKeybinding("a", func() {
		config.AllNamespaces = !config.AllNamespaces
		allText.SetText("changing to " + map[bool]string{true: "all", false: "default"}[config.AllNamespaces] + " ...")
		ui.Repaint()
		refresh()
	})
	ui.SetKeybinding("Up", func() { scrollbox.Scroll(0, -1) })
	ui.SetKeybinding("Down", func() { scrollbox.Scroll(0, 1) })
	ui.SetKeybinding("Left", func() { scrollbox.Scroll(-1, 0) })
	ui.SetKeybinding("Right", func() { scrollbox.Scroll(1, 0) })
	ui.SetKeybinding("t", func() { scrollbox.ScrollToTop() })
	ui.SetKeybinding("b", func() { scrollbox.ScrollToBottom() })
	ui.SetKeybinding("+", func() {
		config.RefreshSeconds++
		refreshText.SetText(fmt.Sprint(config.RefreshSeconds) + "s")
		ui.Repaint()
	})
	ui.SetKeybinding("-", func() {
		if config.RefreshSeconds > 1 {
			config.RefreshSeconds--
			refreshText.SetText(fmt.Sprint(config.RefreshSeconds) + "s")
			ui.Repaint()
		}
	})
	ui.SetKeybinding("n", func() {
		config.StageIndex = (config.StageIndex + 1) % len(config.Stages)
		stageText.SetText("changing to " +
			config.Stages[config.StageIndex].Name + "...")
		ui.Repaint()
		refresh()
	})
	ui.SetKeybinding("p", func() {
		config.StageIndex--
		if config.StageIndex < 0 {
			config.StageIndex = len(config.Stages) - 1
		}
		stageText.SetText("changing to " +
			config.Stages[config.StageIndex].Name + "...")
		ui.Repaint()
		refresh()
	})

	refresh()

	go func() {
		o := 0
		t := time.NewTicker(time.Second)
		for range t.C {
			o++
			if o >= config.RefreshSeconds {
				refresh()
				o = 0
			}
		}
	}()

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
