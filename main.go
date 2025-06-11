package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"image/color"

	"gopkg.in/yaml.v3"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Act struct {
	Name string `yaml:"name"`
	Time string `yaml:"time"`
}

type Config struct {
	Persons []string `yaml:"persons"`
	Random  bool     `yaml:"random"`
	Acts    []Act    `yaml:"acts"`
	Counter bool     `yaml:"counter"`
	Next    bool     `yaml:"next"`
}

var (
	currentPersonIndex = 0
	currentActIndex    = 0
	config             Config
	fontSize           = 25
	timer              *time.Timer
	ticker             *time.Ticker
	paused             bool
	remaining          time.Duration
	currentDuration    time.Duration

	progressBar   *widget.ProgressBar
	totalProgress *widget.ProgressBar
	nextLabel     *canvas.Text
	personLabel   *canvas.Text
	stageLabel    *canvas.Text
	pauseBtn      *widget.Button
)

func loadConfig(path string) Config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("failed to parse yaml: %v", err)
	}
	if cfg.Random {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(cfg.Persons), func(i, j int) {
			cfg.Persons[i], cfg.Persons[j] = cfg.Persons[j], cfg.Persons[i]
		})
	}
	return cfg
}

func updateProgressBar() {
	fraction := remaining.Seconds() / currentDuration.Seconds()
	if fraction < 0 {
		fraction = 0
	}
	progressBar.SetValue(fraction)
	progressBar.TextFormatter = func() string {
		return remaining.String()
	}
	progressBar.Refresh()
}

func updateTotalProgressBar() {
	totalProgress.SetValue(float64(currentPersonIndex+1) / float64(len(config.Persons)))
	totalProgress.TextFormatter = func() string {
		return fmt.Sprintf("%d / %d", currentPersonIndex+1, len(config.Persons))
	}
	totalProgress.Refresh()
}

func updateUI() {
	if currentPersonIndex >= len(config.Persons) {
		personLabel.Text = "Done"
		canvas.Refresh(personLabel)
		return
	}
	personLabel.Text = config.Persons[currentPersonIndex]
	stage := config.Acts[currentActIndex]
	stageLabel.Text = stage.Name
	canvas.Refresh(stageLabel)

	dur, _ := time.ParseDuration(stage.Time)
	remaining = dur
	currentDuration = dur
	updateNextLabel()
	canvas.Refresh(personLabel)
	startTimer(dur)
}

func updateNextLabel() {
	if !config.Next {
		nextLabel.Text = ""
		canvas.Refresh(nextLabel)
		return
	}
	if currentPersonIndex+1 >= len(config.Persons) {
		nextLabel.Text = "Next: Last"
		canvas.Refresh(nextLabel)
	} else {
		nextLabel.Text = "Next: " + config.Persons[currentPersonIndex+1]
		canvas.Refresh(nextLabel)
	}
}

func startTimer(duration time.Duration) {
	if timer != nil {
		timer.Stop()
	}
	paused = false
	if ticker != nil {
		ticker.Stop()
	}
	ticker = time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			if paused {
				continue
			}
			remaining -= time.Second
			if remaining < 0 {
				remaining = 0
			}
			updateProgressBar()
			if remaining <= 0 {
				ticker.Stop()
				nextStage()
				return
			}
		}
	}()
}

func nextStage() {
	currentActIndex++
	if currentActIndex >= len(config.Acts) {
		currentActIndex = 0
		currentPersonIndex++
		updateTotalProgressBar()
	}
	updateUI()
}

func makeUI(a fyne.App, w fyne.Window) fyne.CanvasObject {
	personLabel = canvas.NewText("", nil)
	personLabel.TextSize = float32(fontSize + 10)
	stageLabel = canvas.NewText("", nil)
	stageLabel.TextSize = float32(fontSize + 5)
	stageLabel.Color = color.RGBA{R: 144, G: 238, B: 144, A: 255} // light green

	progressBar = widget.NewProgressBar()
	totalProgress = widget.NewProgressBar()
	totalProgress.Max = 1.0
	nextLabel = canvas.NewText("", nil)
	nextLabel.TextSize = float32(fontSize + 8)
	nextLabel.Color = color.RGBA{R: 255, G: 255, B: 153, A: 255}

	nextBtn := widget.NewButton("Next", func() {
		paused = false
		if ticker != nil {
			ticker.Stop()
		}
		pauseBtn.SetText("Pause")
		nextStage()
	})

	pauseBtn = widget.NewButton("Pause", func() {
		paused = !paused
		if paused {
			pauseBtn.SetText("Resume")
		} else {
			pauseBtn.SetText("Pause")
		}
	})

	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeySpace {
			pauseBtn.OnTapped()
		}
	})
	settingsBtn := widget.NewButton("Settings", func() {
		entry := widget.NewEntry()
		entry.SetText(fmt.Sprintf("%d", fontSize))
		d := dialog.NewForm("Set Font Size", "Apply", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Font Size", entry),
		}, func(b bool) {
			if b {
				if val, err := fmt.Sscanf(entry.Text, "%d", &fontSize); err == nil && val == 1 {
					personLabel.TextSize = float32(fontSize + 10)
					canvas.Refresh(personLabel)
				}
			}
		}, w)
		d.Resize(fyne.NewSize(300, 200))
		d.Show()
	})

	return container.NewVBox(
		personLabel,
		stageLabel,
		progressBar,
		totalProgress,
		nextLabel,
		container.NewHBox(nextBtn, pauseBtn, settingsBtn),
	)
}

type redTheme struct {
	base fyne.Theme
}

func (r redTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == fyne.ThemeColorName("progressBarFilled") {
		return color.RGBA{R: 255, G: 0, B: 0, A: 255}
	}
	return r.base.Color(name, variant)
}

func (r redTheme) Font(style fyne.TextStyle) fyne.Resource {
	return r.base.Font(style)
}

func (r redTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return r.base.Icon(name)
}

func (r redTheme) Size(name fyne.ThemeSizeName) float32 {
	return r.base.Size(name)
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("YAML file path must be provided as the first argument")
	}
	config = loadConfig(args[0])
	a := app.New()
	a.Settings().SetTheme(redTheme{base: a.Settings().Theme()})
	w := a.NewWindow("Timer GUI")
	w.Resize(fyne.NewSize(600, 300))

	ui := makeUI(a, w)
	w.SetContent(ui)

	updateUI()
	updateTotalProgressBar()
	w.ShowAndRun()
}
