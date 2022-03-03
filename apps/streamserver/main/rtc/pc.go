package rtc

import (
	"fmt"
	"os"
	"server/main/utils"

	"github.com/go-resty/resty/v2"
	"github.com/olebedev/emitter"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type PeerConnection struct {
	ViewerId       string
	OnSignal       func(cb func(signal Signal))
	Signal         func(signal Signal) error
	OnConnected    func(cb func())
	OnDisconnected func(cb func())
	AttachTracks   func(tracks *StreamTracks)
	*webrtc.PeerConnection
	*emitter.Emitter
}
type ICEServer struct {
	URLs           []string    `json:"urls"`
	Username       string      `json:"username,omitempty"`
	Credential     interface{} `json:"credential,omitempty"`
	CredentialType string      `json:"credentialType,omitempty"`
}

func (peerConnection *PeerConnection) initializeConnection() {
	signalingServer, hasEnv := os.LookupEnv("SIGNAL_SERVER_URL")
	if !hasEnv {
		panic("SIGNAL_SERVER_URL not set")
	}

	//Get ice server config from the signalserver
	client := resty.New()
	res, _ := client.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/ice-config", signalingServer))

	parsed := utils.ParseJson[[]ICEServer](res)
	iceServers := parsed.Value

	parsedServers := make([]webrtc.ICEServer, len(iceServers))
	for i, iceServer := range iceServers {
		parsedServers[i] = webrtc.ICEServer{
			URLs:           iceServer.URLs,
			Username:       iceServer.Username,
			Credential:     iceServer.Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		}
	}
	log.Info().Msgf("Got ice servers: %+v", parsedServers)

	_peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: parsedServers,
	})

	if err != nil {
		panic(err)
	}

	peerConnection.PeerConnection = _peerConnection

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		json := c.ToJSON()
		signal := Signal{
			ViewerId:  peerConnection.ViewerId,
			Type:      "candidate",
			Candidate: json,
		}
		peerConnection.Emit("signal", signal)
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Info().
			Str("state", connectionState.String()).
			Str("viewerId", peerConnection.ViewerId).
			Msg("ICE Connection State has changed")
	})

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		if connectionState == webrtc.PeerConnectionStateDisconnected ||
			connectionState == webrtc.PeerConnectionStateClosed ||
			connectionState == webrtc.PeerConnectionStateFailed {
			peerConnection.Close()
			peerConnection.Emit("disconnected")
		} else if connectionState == webrtc.PeerConnectionStateConnected {
			peerConnection.Emit("connected")
		}
	})
}

func (peerConnection *PeerConnection) applyOffer(signal Signal) (*Signal, error) {

	if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{SDP: signal.SDP, Type: webrtc.SDPTypeOffer}); err != nil {
		log.Err(err).Send()
		return nil, err
	}

	answer, answerErr := peerConnection.CreateAnswer(nil)
	if answerErr != nil {
		log.Err(answerErr).Send()
		return nil, answerErr
	}
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		log.Err(err).Send()
		return nil, err
	}

	answerSignal := &Signal{
		Type:     "answer",
		ViewerId: peerConnection.ViewerId,
		SDP:      answer.SDP,
	}

	return answerSignal, nil

}

func newConnection(viewerId string) (peerConnection *PeerConnection) {
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)

	initialized := false
	peerConnection = &PeerConnection{
		Emitter:  e,
		ViewerId: viewerId,
		Signal: func(signal Signal) error {
			switch signal.Type {
			case "offer":
				answerSignal, err := peerConnection.applyOffer(signal)
				if err != nil {
					log.Err(err).Send()
					return err
				}
				e.Emit("signal", *answerSignal)
				initialized = true

			case "candidate":
				if !initialized {
					log.Warn().
						Str("viewerId", signal.ViewerId).
						Msg("Received candidate before offer, ignoring")

					return fmt.Errorf("Received candidate before offer")
				}
				if err := peerConnection.AddICECandidate(signal.Candidate); err != nil {
					log.Err(err).Send()
					return err
				}

			}
			return nil

		},
		OnSignal: func(cb func(signal Signal)) {
			e.On("signal", func(e *emitter.Event) {
				cb(e.Args[0].(Signal))
			})
		},
		OnDisconnected: func(cb func()) {
			peerConnection.Once("disconnected", func(e *emitter.Event) {
				cb()
			})
		},
		OnConnected: func(cb func()) {
			peerConnection.Once("connected", func(e *emitter.Event) {
				cb()
			})
		},
		AttachTracks: func(tracks *StreamTracks) {
			rtpSender, err := peerConnection.AddTrack(tracks.AudioTrack)
			if err != nil {
				panic(err)
			}
			processRTCP(rtpSender)

			rtpSender, err = peerConnection.AddTrack(tracks.VideoTrack)
			if err != nil {
				panic(err)
			}
			processRTCP(rtpSender)
		},
		PeerConnection: nil,
	}

	//This will set the peerConnection.PeerConnection
	peerConnection.initializeConnection()

	return peerConnection
}
