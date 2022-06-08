package remote

import (
	"capture/main/rtc"
	"capture/main/utils"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/olebedev/emitter"
	"github.com/pion/webrtc/v3"
	hook "github.com/robotn/gohook"
	"github.com/rs/zerolog/log"
)

type Command struct {
	Type   string  `json:"type"`
	NormX  float32 `json:"normX"`
	NormY  float32 `json:"normY"`
	Button int     `json:"button"`
	Key    string  `json:"key"`
	Delta  float32 `json:"delta"`
}

var mouse_keys = map[int]string{
	0: "left",
	1: "middle",
	2: "right",
}

var capturing = false

func captureCursor(e *emitter.Emitter) {

	// 60 fps ticker
	ticker := time.NewTicker(time.Second / 60)
	for range ticker.C {
		x, y := robotgo.GetMousePos()
		screen_x, screen_y := GetScreenSizes()
		norm_x := float32(x) / float32(screen_x)
		norm_y := float32(y) / float32(screen_y)
		command := Command{
			Type:  "s_move",
			NormX: norm_x,
			NormY: norm_y,
		}

		data, err := json.Marshal(command)
		if err != nil {
			panic(err)
		}

		e.Emit("output", data)

	}

}

func captureClicks(e *emitter.Emitter) {

	hook.Register(hook.MouseHold, []string{}, func(ev hook.Event) {
		if ev.Button == hook.MouseMap["left"] {
			command := Command{
				Type: "s_mousedown",
			}

			data, err := json.Marshal(command)
			if err != nil {
				panic(err)
			}

			e.Emit("output", data)
		}
	})

	hook.Register(hook.MouseDown, []string{}, func(ev hook.Event) {
		if ev.Button == hook.MouseMap["left"] {

			command := Command{
				Type: "s_mouseup",
			}

			data, err := json.Marshal(command)
			if err != nil {
				panic(err)
			}

			e.Emit("output", data)
		}
	})

}

func clamp(val int, min int, max int) int {
	if val < min {
		return min
	}

	if val > max {
		return max
	}

	return val
}

func handleSpecialKey(key string) string {
	var mapped_key string
	lower_key := strings.ToLower(key)

	switch lower_key {
	case "arrowdown":
		mapped_key = "down"
	case "arrowup":
		mapped_key = "up"
	case "arrowleft":
		mapped_key = "left"
	case "arrowright":
		mapped_key = "right"
	case "altgraph":
		mapped_key = "ralt"
	default:
		mapped_key = key
	}
	return mapped_key
}

func GetScreenSizes() (int, int) {
	config := utils.GetConfig()
	screen_original_x, screen_y := robotgo.GetScreenSize()

	// determine screen_x using the height, because robotgo desn't support multiple monitors properly
	height_scale := float32(screen_y) / float32(config.ResolutionY)

	screen_x := clamp(int(float32(config.ResolutionX)*height_scale), 0, screen_original_x)

	return screen_x, screen_y
}

func ProcessControlCommands(e *emitter.Emitter) {
	log.Info().Msg("Starting control commands handler")
	screen_x, screen_y := GetScreenSizes()

	e.On("input", func(e *emitter.Event) {
		data := e.Args[0].([]byte)

		var command Command
		err := json.Unmarshal(data, &command)
		if err != nil {
			panic(err)
		}

		if command.Type == "move" {
			x := clamp(int(command.NormX*float32(screen_x)), 0, screen_x)
			y := clamp(int(command.NormY*float32(screen_y)), 0, screen_y)
			// fmt.Printf("Received mouse command: %d, %d \n", x, y)
			robotgo.Move(int(x), int(y))
		}

		if command.Type == "mousedown" {
			mouse_key := mouse_keys[command.Button]
			fmt.Printf("Received mouse down command: %s \n", mouse_key)
			robotgo.Toggle(mouse_key, "down")
		}

		if command.Type == "mouseup" {
			mouse_key := mouse_keys[command.Button]
			fmt.Printf("Received mouse up command: %s \n", mouse_key)
			robotgo.Toggle(mouse_key, "up")
		}
		if command.Type == "keydown" {
			mapped_key := handleSpecialKey(command.Key)
			fmt.Printf("Received keydown: %s \n", mapped_key)
			robotgo.KeyDown(mapped_key)
		}
		if command.Type == "keyup" {
			mapped_key := handleSpecialKey(command.Key)
			fmt.Printf("Received keyup: %s \n", mapped_key)
			robotgo.KeyUp(mapped_key)
		}
		if command.Type == "wheel" {
			fmt.Printf("Received wheel: %f \n", command.Delta)
			robotgo.Scroll(0, clamp(-int(command.Delta), -1, 1))
		}

	})

}

func SetupRemote(peerConnection *rtc.PeerConnection) {
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)
	config := utils.GetConfig()

	if config.RemoteEnabled {
		ProcessControlCommands(e)
	}

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			if !capturing {
				capturing = true
				go captureClicks(e)
				go captureCursor(e)

				go func() {
					s := hook.Start()
					<-hook.Process(s)
				}()

			}
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			e.Emit("input", msg.Data)
		})

		e.On("output", func(event *emitter.Event) {

			data := event.Args[0].([]byte)

			//check if the channel is open
			if dc.ReadyState() != webrtc.DataChannelStateOpen {
				return
			}

			dc.Send(data)
		})

	})
}
