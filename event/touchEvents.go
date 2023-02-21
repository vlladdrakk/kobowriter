package event

import (
	"context"
	"fmt"
	"log"

	"github.com/asaskevich/EventBus"
	"github.com/kenshaw/evdev"
)

const (
	Untouched = -1
	Pressed   = 1
	Released  = 0
)

type TouchEvent struct {
	X       int32
	Y       int32
	Pressed int // -1 = nil, 0 = false, 1 = true
}

func (t *TouchEvent) Init() {
	t.X = -1
	t.Y = -1
	t.Pressed = -1
}

func (t *TouchEvent) checkPressed(value int32) {
	if value > 0 {
		t.Pressed = 1
	} else {
		t.Pressed = 0
	}
}

func (t *TouchEvent) complete() bool {
	return t.X != -1 && t.Y != -1 && t.Pressed != -1
}

func (t *TouchEvent) toString() string {
	return fmt.Sprintf("%d:%d:%d", t.X, t.Y, t.Pressed)
}

// Monitor for touch events and send events on the bus
func TouchEventLoop(devicePath string, bus EventBus.Bus) {
	// open
	deviceFile, err := evdev.OpenFile(devicePath)
	if err != nil {
		log.Fatal(err)
	}
	defer deviceFile.Close()

	// start polling
	ch := deviceFile.Poll(context.Background())

	var tEvent TouchEvent

	tEvent.Init()

loop:
	for {
		select {
		case event := <-ch:
			// channel closed
			if event == nil {
				break loop
			}

			switch event.Type.(type) {
			case evdev.AbsoluteType:
				switch event.Type {
				case evdev.AbsoluteX:
					tEvent.X = event.Value
				case evdev.AbsoluteY:
					tEvent.Y = event.Value
				case evdev.AbsolutePressure:
					tEvent.checkPressed(event.Value)
				}

				if tEvent.complete() {
					bus.Publish("TOUCH_EVENT", tEvent.toString())

					// If the touch has ended, reset the event
					if tEvent.Pressed == 0 {
						tEvent.Init()
					}
				}
			}
		}
	}
}
