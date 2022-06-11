package rtc

import (
	"bytes"
	"fmt"
	"time"

	"github.com/olebedev/emitter"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Signal struct {
	ViewerId  string                  `json:"viewerId"`
	Type      string                  `json:"type"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	SDP       string                  `json:"sdp"`
}
type Tracks struct {
	VideoTrack *webrtc.TrackLocalStaticSample
	AudioTrack *webrtc.TrackLocalStaticSample
}

type PeerConnection struct {
	Id                string
	OnSignal          func(cb func(signal Signal))
	Signal            func(signal Signal) error
	OnConnected       func(cb func())
	OnDisconnected    func(cb func())
	AddTracks         func(tracks *Tracks)
	ConnectTo         func(peerConnection *PeerConnection)
	LocalTracks       []*webrtc.TrackLocalStaticRTP
	PendingCandidates []*webrtc.ICECandidate
	SetSnapshot       func(snapshot *bytes.Buffer)
	GetSnapshot       func() *bytes.Buffer
	DataChannel       *webrtc.DataChannel
	*webrtc.PeerConnection
	*emitter.Emitter

	EmitterVoid *emitter.Emitter
}
type ICEServer struct {
	URLs           []string    `json:"urls"`
	Username       string      `json:"username,omitempty"`
	Credential     interface{} `json:"credential,omitempty"`
	CredentialType string      `json:"credentialType,omitempty"`
}

func (peerConnection *PeerConnection) initializeConnection() {
	iceServers := GetRtcConfig().ICEServers
	parsedServers := make([]webrtc.ICEServer, len(iceServers))
	for i, iceServer := range iceServers {
		parsedServers[i] = webrtc.ICEServer{
			URLs:           iceServer.URLs,
			Username:       iceServer.Username,
			Credential:     iceServer.Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		}
	}
	log.Info().Msgf("Ice servers from config: %+v", parsedServers)
	//api := SetupApi()
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
		desc := peerConnection.RemoteDescription()
		if desc == nil {
			peerConnection.PendingCandidates = append(peerConnection.PendingCandidates, c)
			return
		}

		json := c.ToJSON()
		signal := Signal{
			ViewerId:  peerConnection.Id,
			Type:      "candidate",
			Candidate: json,
		}
		peerConnection.Emit("signal", signal)
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Info().
			Str("state", connectionState.String()).
			Str("viewerId", peerConnection.Id).
			Msg("ICE Connection State has changed")
	})

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		if connectionState == webrtc.PeerConnectionStateDisconnected {
			peerConnection.EmitterVoid.Emit("disconnected")

			peerConnection.Close()
		} else if connectionState == webrtc.PeerConnectionStateConnected {
			peerConnection.EmitterVoid.Emit("connected")

		}
	})

	peerConnection.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		localTrack := peerConnection.AddRemoteTrack(tr)
		peerConnection.LocalTracks = append(peerConnection.LocalTracks, localTrack)
		go func() {
			peerConnection.Emit("track", localTrack)
		}()
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
		ViewerId: peerConnection.Id,
		SDP:      answer.SDP,
	}

	return answerSignal, nil

}

type Packet struct {
	len int
	buf []byte
}

func (peerConnection *PeerConnection) AddRemoteTrack(tr *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	outputTrack, err := webrtc.NewTrackLocalStaticRTP(tr.Codec().RTPCodecCapability, tr.ID(), "proxy")
	if err != nil {
		panic(err)
	}
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for range ticker.C {
			if peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				return
			}

			errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(tr.SSRC())}})
			if errSend != nil {
				return
			}
		}
	}()

	go func() {
		rtp_buffer := make(chan *Packet, 1024)

		defer func() {
			close(rtp_buffer)
		}()

		// consumer
		go func() {
			for {
				if peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
					return
				}

				packet, ok := <-rtp_buffer

				if !ok {
					return
				}

				if _, writeErr := outputTrack.Write(packet.buf[:packet.len]); writeErr != nil {
					return
				}

			}
		}()

		// producer
		for {
			if peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				return
			}

			b := make([]byte, 1460)
			i, _, err := tr.Read(b)
			if err != nil {
				return
			}

			packet := &Packet{
				len: i,
				buf: b,
			}
			rtp_buffer <- packet

		}

	}()

	return outputTrack
}

func (peerConnection *PeerConnection) Initiate() {

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	signal := Signal{
		ViewerId: peerConnection.Id,
		Type:     "offer",
		SDP:      offer.SDP,
	}

	go peerConnection.Emit("signal", signal)

}

func connectDatachannel(a *PeerConnection, b *PeerConnection) {
	if a.DataChannel != nil {
		a.DataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			a.Emit("almafa")
			if b.DataChannel != nil {
				b.DataChannel.Send(msg.Data)
			}
		})
	} else {
		a.OnDataChannel(func(dc *webrtc.DataChannel) {
			a.DataChannel = dc
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				a.Emit("almafa")
				if b.DataChannel != nil {
					b.DataChannel.Send(msg.Data)
				}
			})
		})
	}
}

func newConnection(Id string) (peerConnection *PeerConnection) {
	e := &emitter.Emitter{}
	eVoid := &emitter.Emitter{}

	eVoid.Use("*", emitter.Void)
	localTracks := make([]*webrtc.TrackLocalStaticRTP, 0)
	pendingCandidates := make([]*webrtc.ICECandidate, 0)
	var snapshot *bytes.Buffer = nil

	initialized := false
	peerConnection = &PeerConnection{
		EmitterVoid:       eVoid,
		PendingCandidates: pendingCandidates,
		LocalTracks:       localTracks,
		Emitter:           e,
		Id:                Id,
		Signal: func(signal Signal) error {
			switch signal.Type {
			case "offer":
				initialized = true
				answerSignal, err := peerConnection.applyOffer(signal)
				if err != nil {
					log.Err(err).Send()
					return err
				}
				go func() {
					e.Emit("signal", *answerSignal)
				}()
			case "answer":
				initialized = true
				if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{SDP: signal.SDP, Type: webrtc.SDPTypeAnswer}); err != nil {
					log.Err(err).Send()
					return err
				}
				for _, c := range peerConnection.PendingCandidates {
					json := c.ToJSON()
					signal := Signal{
						ViewerId:  peerConnection.Id,
						Type:      "candidate",
						Candidate: json,
					}
					go func() {
						e.Emit("signal", signal)
					}()
				}
			case "candidate":
				if !initialized {
					log.Warn().
						Str("viewerId", signal.ViewerId).
						Msg("server received candidate before offer, ignoring")

					return fmt.Errorf("received candidate before offer")
				}
				if err := peerConnection.AddICECandidate(signal.Candidate); err != nil {
					log.Err(err).Send()
					return err
				}
			}
			return nil
		},
		OnSignal: func(cb func(signal Signal)) {
			go func() {
				for event := range e.On("signal") {
					go func(event emitter.Event) {
						defer func() {
							if r := recover(); r != nil {
								log.Error().Msg("panic in OnSignal")
							}
						}()
						cb(event.Args[0].(Signal))
					}(event)

				}
			}()
		},
		OnDisconnected: func(cb func()) {
			eVoid.On("disconnected", func(e *emitter.Event) {
				cb()
			})
		},
		OnConnected: func(cb func()) {
			eVoid.On("connected", func(e *emitter.Event) {
				cb()
			})

		},
		ConnectTo: func(other *PeerConnection) {

			connectDatachannel(peerConnection, other)
			connectDatachannel(other, peerConnection)

			if peerConnection.ConnectionState() == webrtc.PeerConnectionStateConnected {
				if len(peerConnection.LocalTracks) > 0 {
					for _, track := range peerConnection.LocalTracks {
						other.AddTrack(track)
					}
				}
			} else {
				peerConnection.OnConnected(func() {
					time.Sleep(time.Millisecond * 1000)
					if len(peerConnection.LocalTracks) > 0 {
						for _, track := range peerConnection.LocalTracks {
							_, err := other.AddTrack(track)
							if err != nil {
								panic(err)
							}
							//processRTCP(rtpSender)
						}
						time.Sleep(time.Millisecond * 1000)
						// re-negotiate with the browser
						other.Initiate()

					}
					peerConnection.On("track", func(e *emitter.Event) {
						track := e.Args[0].(*webrtc.TrackLocalStaticRTP)
						_, err := other.AddTrack(track)
						if err != nil {
							panic(err)
						}
						//processRTCP(rtpSender)
						// re-negotiate with the browser
						other.Initiate()

					})
				})
			}

		},
		SetSnapshot: func(_snapshot *bytes.Buffer) {
			snapshot = _snapshot
		},
		GetSnapshot: func() *bytes.Buffer {
			return snapshot
		},
		PeerConnection: nil,
	}

	//This will set the peerConnection.PeerConnection
	peerConnection.initializeConnection()

	return peerConnection
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
