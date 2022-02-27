package stream

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"server/main/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/go-vgo/robotgo"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
)

type Signal struct {
	ViewerId  string                  `json:"viewerId"`
	Type      string                  `json:"type"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	SDP       string                  `json:"sdp"`
}

func randomStr() string {
	n := 5
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}

//A map to store connections by their ID
var connections = make(map[string]*webrtc.PeerConnection)

//collect signals
var outgoing_signal_chan = make(chan Signal, 100)

func SetupNewConnection(getVideoChannelFn func() chan *gst.Buffer, getAudioChannelFn func() chan *gst.Buffer, viewerId string) (peerConnection *webrtc.PeerConnection) {

	iceServers := make([]webrtc.ICEServer, 0)

	turn_server_url, hasEnv := os.LookupEnv("TURN_SERVER_URL")
	if hasEnv && turn_server_url != "" {
		iceServer := webrtc.ICEServer{
			URLs: []string{turn_server_url},
		}
		turn_server_username := os.Getenv("TURN_SERVER_USERNAME")
		if turn_server_username != "" {
			iceServer.Username = turn_server_username
		}
		turn_server_password := os.Getenv("TURN_SERVER_PASSWORD")
		if turn_server_password != "" {
			iceServer.Credential = turn_server_password
			iceServer.CredentialType = webrtc.ICECredentialTypePassword
		}
		iceServers = append(iceServers, iceServer)
	}

	stun_server_url, hasEnv := os.LookupEnv("STUN_SERVER_URL")
	if hasEnv && stun_server_url != "" {
		iceServer := webrtc.ICEServer{
			URLs: []string{stun_server_url},
		}
		stun_server_username := os.Getenv("STUN_SERVER_USERNAME")
		if stun_server_username != "" {
			iceServer.Username = stun_server_username
		}
		stun_server_password := os.Getenv("STUN_SERVER_PASSWORD")
		if stun_server_password != "" {
			iceServer.Credential = stun_server_password
			iceServer.CredentialType = webrtc.ICECredentialTypePassword
		}
		iceServers = append(iceServers, iceServer)
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: iceServers,
	})

	if err != nil {
		panic(err)
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "video", "pion")
	if err != nil {
		panic(err)
	}

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}
	processRTCP(rtpSender)

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}, "audio", "pion")
	if err != nil {
		panic(err)
	}

	rtpSender, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}
	processRTCP(rtpSender)

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		json := c.ToJSON()
		signal := Signal{
			ViewerId:  viewerId,
			Type:      "candidate",
			Candidate: json,
		}
		outgoing_signal_chan <- signal
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Info().Str("state", connectionState.String()).Msg("ICE Connection State has changed")
	})

	sendVideo := func() {
		channel := getVideoChannelFn()
		for frame_buffer := range channel {
			if peerConnection.ConnectionState() == webrtc.PeerConnectionStateDisconnected {
				return
			}
			copied := frame_buffer.DeepCopy()
			err = videoTrack.WriteSample(media.Sample{Data: copied.Bytes(), Duration: copied.Duration()})
			if err != nil {
				log.Err(err).Send()
				return
			}
		}
	}

	sendAudio := func() {
		channel := getAudioChannelFn()
		for sample_buffer := range channel {
			if peerConnection.ConnectionState() == webrtc.PeerConnectionStateDisconnected {
				return
			}
			copied := sample_buffer.DeepCopy()
			err = audioTrack.WriteSample(media.Sample{Data: copied.Bytes(), Duration: copied.Duration()})
			if err != nil {
				log.Err(err).Send()
				return
			}
		}
	}

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		if connectionState == webrtc.PeerConnectionStateConnected {
			go sendVideo()
			go sendAudio()
		}

	})

	return peerConnection
}

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

func SetupRemoteControl(peerConnection *webrtc.PeerConnection) {

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		screen_x, screen_y := robotgo.GetScreenSize()
		d.OnOpen(func() {
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
	})
}

func StartWrtcServer() {

	signalingServer, hasEnv := os.LookupEnv("SIGNAL_SERVER_URL")
	if !hasEnv {
		panic("SIGNAL_SERVER_URL not set")
	}

	streamId, hasEnv := os.LookupEnv("STREAM_ID")
	if !hasEnv {
		panic("STREAM_ID not set")
	}

	remoteEnabled, hasEnv := os.LookupEnv("REMOTE_ENABLED")
	if !hasEnv {
		remoteEnabled = "false"
	}

	remoteEnabledBool, err := strconv.ParseBool(remoteEnabled)
	if err != nil {
		log.Err(err).Msg("Failed to parse REMOTE_ENABLED")
		remoteEnabledBool = false
	}

	log.Info().Str("STREAM_ID", streamId).Send()

	getVideoChannelFn := CreateVideoCapture()
	getAudioChannelFn := CreateAudioCapture()
	go func() {
		client := resty.New()
		for {
			res, err := client.R().
				SetHeader("Accept", "application/json").
				Get(fmt.Sprintf("%s/signal/%s/internal", signalingServer, streamId))

			if err != nil {
				log.Err(err).Send()
				time.Sleep(time.Second * 1)
				continue
			}

			body := utils.ParseJson[[]Signal](res)

			//offers come before candidates
			sortedSignals := make([]Signal, 0)
			for _, signal := range body.Value {
				if signal.Type == "offer" {
					sortedSignals = append(sortedSignals, signal)
				}
			}
			for _, signal := range body.Value {
				if signal.Type == "candidate" {
					sortedSignals = append(sortedSignals, signal)
				}
			}

			log.Info().Int("count", len(sortedSignals)).Msg("Received signals")

			for _, signal := range sortedSignals {

				viewerId := signal.ViewerId
				peerConnection := connections[viewerId]

				if peerConnection == nil {
					if signal.Type == "offer" {
						peerConnection = SetupNewConnection(getVideoChannelFn, getAudioChannelFn, viewerId)
						if remoteEnabledBool {
							SetupRemoteControl(peerConnection)
						}
						connections[viewerId] = peerConnection

					} else {
						continue
					}
				}

				if signal.Type == "candidate" {
					if err := peerConnection.AddICECandidate(signal.Candidate); err != nil {
						log.Err(err).Send()
						return
					}
				}

				if signal.Type == "offer" {

					if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{SDP: signal.SDP, Type: webrtc.SDPTypeOffer}); err != nil {
						log.Err(err).Send()
						return
					}
					answer, answerErr := peerConnection.CreateAnswer(nil)
					if answerErr != nil {
						log.Err(answerErr).Send()
						return
					}
					if err := peerConnection.SetLocalDescription(answer); err != nil {
						log.Err(err).Send()
						return
					}
					signal := Signal{
						Type:     "answer",
						ViewerId: viewerId,
						SDP:      answer.SDP,
					}
					outgoing_signal_chan <- signal
				}
			}
			time.Sleep(time.Second * 1)
		}
	}()

	go func() {
		client := resty.New()
		signals_to_send := make([]Signal, 0)
		for {
			select {
			case signal := <-outgoing_signal_chan:
				signals_to_send = append(signals_to_send, signal)
			case <-time.After(time.Millisecond * 100):
				if len(signals_to_send) > 0 {
					to_send := signals_to_send
					signals_to_send = make([]Signal, 0)
					client.R().
						SetBody(to_send).
						Post(fmt.Sprintf("%s/signal/%s/internal", signalingServer, streamId))
				}
			}
		}
	}()

}

func processRTCP(rtpSender *webrtc.RTPSender) {
	go func() {
		rtcpBuf := make([]byte, 1500)

		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()
}
