package event

import (
	"context"
	"fmt"
	"log"

	"github.com/asaskevich/EventBus"
	"github.com/kenshaw/evdev"
)

type touchEvent struct {
	x       int32
	y       int32
	pressed int // -1 = nil, 0 = false, 1 = true
}

func (t *touchEvent) init() {
	t.x = -1
	t.y = -1
	t.pressed = -1
}

func (t *touchEvent) checkPressed(value int32) {
	if value > 0 {
		t.pressed = 1
	} else {
		t.pressed = 0
	}
}

func (t *touchEvent) complete() bool {
	return t.x != -1 && t.y != -1 && t.pressed != -1
}

func (t *touchEvent) toString() string {
	return fmt.Sprintf("%d:%d:%d", t.x, t.y, t.pressed)
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

	var tEvent touchEvent

	tEvent.init()

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
					tEvent.x = event.Value
				case evdev.AbsoluteY:
					tEvent.y = event.Value
				case evdev.AbsolutePressure:
					tEvent.checkPressed(event.Value)
				}

				if tEvent.complete() {
					bus.Publish("TOUCH_EVENT", tEvent.toString())

					// If the touch has ended, reset the event
					if tEvent.pressed == 0 {
						tEvent.init()
					}
				}
			}
		}
	}
}
