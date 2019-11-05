package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	tui "github.com/marcusolsson/tui-go"
)

type Stage struct {
	Name       string `json:"name"`
	ConfigFile string `json:"configFile"`
}

func main() {
	root := tui.NewHBox()

	var config struct {
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

	getPodsText := tui.NewLabel("")
	topNodeText := tui.NewLabel("")
	topPodText := tui.NewLabel("")
	refreshText := tui.NewLabel(fmt.Sprint(config.RefreshSeconds) + "s")
	stageText := tui.NewLabel(config.Stages[config.StageIndex].Name)

	getPodsBox := tui.NewVBox(getPodsText)
	getPodsBox.SetBorder(true)

	topNodeBox := tui.NewVBox(topNodeText)
	topNodeBox.SetBorder(true)

	topPodTextBox := tui.NewVBox(topPodText)
	topPodTextBox.SetBorder(true)

	infoBox := tui.NewHBox(refreshText, stageText)
	infoBox.SetBorder(true)

	v := tui.NewVBox(
		infoBox,
		getPodsBox,
		topNodeBox,
		topPodTextBox,
	)
	scrollbox := tui.NewScrollArea(v)

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

		buf.Reset()
		cmd := exec.Command("kubectl", "get", "pods", "--kubeconfig", config.Stages[config.StageIndex].ConfigFile)
		cmd.Stdout = &buf
		if err := cmd.Run(); err != nil {
			return
		}
		getPodsText.SetText(buf.String())

		buf.Reset()
		cmd = exec.Command("kubectl", "top", "node", "--kubeconfig", config.Stages[config.StageIndex].ConfigFile)
		cmd.Stdout = &buf
		if err := cmd.Run(); err != nil {
			return
		}
		topNodeText.SetText(buf.String())

		buf.Reset()
		cmd = exec.Command("kubectl", "top", "pod", "--kubeconfig", config.Stages[config.StageIndex].ConfigFile)
		cmd.Stdout = &buf
		if err := cmd.Run(); err != nil {
			return
		}
		topPodText.SetText(buf.String())
		ui.Repaint()
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })
	ui.SetKeybinding("c", func() { ui.Quit() })
	ui.SetKeybinding("q", func() { ui.Quit() })
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
		if config.RefreshSeconds > 0 {
			config.RefreshSeconds--
			refreshText.SetText(fmt.Sprint(config.RefreshSeconds) + "s")
			ui.Repaint()
		}
	})
	ui.SetKeybinding("n", func() {
		config.StageIndex = (config.StageIndex + 1) % len(config.Stages)
		stageText.SetText(config.Stages[config.StageIndex].Name)
		ui.Repaint()
		refresh()
	})
	ui.SetKeybinding("p", func() {
		config.StageIndex--
		if config.StageIndex < 0 {
			config.StageIndex = len(config.Stages) - 1
		}
		stageText.SetText(config.Stages[config.StageIndex].Name)
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
