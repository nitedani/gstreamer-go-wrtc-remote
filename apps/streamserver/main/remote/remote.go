package remote

import (
	"encoding/json"
	"fmt"
	"server/main/rtc"
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

func SetupRemoteControl(peerConnection *rtc.PeerConnection) {

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		screen_x, screen_y := robotgo.GetScreenSize()
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

				x := int(command.NormX * float32(screen_x))
				y := int(command.NormY * float32(screen_y))

				//print
				//fmt.Printf("Received mouse command: %d, %d \n", x, y)

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
				fmt.Printf("Received keydown: %s \n", command.Key)
				robotgo.KeyDown(strings.ToLower(command.Key))
			}

			if command.Type == "keyup" {
				fmt.Printf("Received keyup: %s \n", command.Key)
				robotgo.KeyUp(strings.ToLower(command.Key))
			}

			if command.Type == "wheel" {
				fmt.Printf("Received wheel: %f \n", command.Delta)
				robotgo.Scroll(0, int(command.Delta/5))
			}

		})
	})
}
