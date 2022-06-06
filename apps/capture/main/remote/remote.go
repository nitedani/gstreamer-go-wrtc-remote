package remote

import (
	"capture/main/rtc"
	"capture/main/utils"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-vgo/robotgo"
	"github.com/pion/webrtc/v3"
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

func SetupRemoteControl(peerConnection *rtc.PeerConnection) {

	config := utils.GetConfig()

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		screen_original_x, screen_y := robotgo.GetScreenSize()

		// determine screen_x using the height, because robotgo desn't support multiple monitors properly
		height_scale := float32(screen_y) / float32(config.ResolutionY)

		screen_x := clamp(int(float32(config.ResolutionX)*height_scale), 0, screen_original_x)

		d.OnOpen(func() {
			//Send messages here
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {

			var command Command
			err := json.Unmarshal(msg.Data, &command)
			if err != nil {
				panic(err)
			}

			if command.Type == "move" {

				x := clamp(int(command.NormX*float32(screen_x)), 0, screen_x)
				y := clamp(int(command.NormY*float32(screen_y)), 0, screen_y)

				//print
				fmt.Printf("Received mouse command: %d, %d \n", x, y)

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
	})
}
