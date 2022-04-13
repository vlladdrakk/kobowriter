package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"

	"github.com/asaskevich/EventBus"

	_ "embed"

	"github.com/olup/kobowriter/screener"
	"github.com/olup/kobowriter/utils"
	"github.com/olup/kobowriter/views"
)

var saveLocation = "/mnt/onboard/.adds/kobowriter"
var filename = "autosave.txt"

func main() {
	log.Println("Program started")

	// kill all nickel related stuff. Will need a reboot to find back the usual
	log.Println("Killing XCSoar programs ...")
	exec.Command("killall", "-s", "SIGKILL", "KoboMenu").Run()
	exec.Command("killall", "-s", "SIGKILL", "sickel").Run()
	exec.Command("killall", "-s", "SIGKILL", "nickel").Run()

	// rotate screen
	log.Println("Rotate screen ...")
	exec.Command(`fbdepth`, `--rota`, `2`).Run()

	// initialise fbink
	log.Println("Init FBInk ...")

	config := utils.LoadConfig(saveLocation)
	screen := screener.InitScreen(config.FontScale)
	defer screen.Clean()

	bus := EventBus.New()

	c := make(chan bool)
	defer close(c)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		for _ = range sigChan {
			// sig is a ^C, handle it
			screen.ClearFlash()
			c <- true
		}
	}()

	bus.SubscribeAsync("REQUIRE_KEYBOARD", func() {
		config := utils.LoadConfig(saveLocation)
		findKeyboard(screen, bus, config.KeyboardLang)
	}, false)

	bus.SubscribeAsync("QUIT", func() {
		screen.PrintAlert("Good Bye !", 500)

		// quitting
		c <- true
		return
	}, false)

	var unmount func()
	bus.SubscribeAsync("ROUTING", func(routeName string) {
		if unmount != nil {
			unmount()
		}

		switch routeName {
		case "document":
			config := utils.LoadConfig(saveLocation)
			unmount = views.Document(screen, bus, config.LastOpenedDocument)
		case "menu":
			unmount = views.MainMenu(screen, bus, saveLocation)
		case "file-menu":
			unmount = views.FileMenu(screen, bus, saveLocation)
		case "settings-menu":
			unmount = views.SettingsMenu(screen, bus, saveLocation)
		case "language-menu":
			unmount = views.LanguageMenu(screen, bus, saveLocation)
		case "font-menu":
			unmount = views.FontMenu(screen, bus, saveLocation)
		case "app-menu":
			unmount = views.AppMenu(screen, bus, saveLocation)
		case "gemini":
			unmount = views.LaunchGemini(screen, bus, "gemini://gemini.circumlunar.space", saveLocation)
		case "qr":
			unmount = views.Qr(screen, bus, saveLocation)

		default:
			unmount = views.Document(screen, bus, "")
		}

	}, false)

	// init
	bus.Publish("REQUIRE_KEYBOARD")

	for quit := range c {
		if quit {
			break
		}
	}

	println("yo")

}
