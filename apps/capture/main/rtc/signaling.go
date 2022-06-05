package rtc

import (
	"bytes"
	"capture/main/utils"
	"fmt"
	"image/jpeg"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/go-vgo/robotgo"
	"github.com/nfnt/resize"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Signal struct {
	ViewerId  string                  `json:"viewerId"`
	Type      string                  `json:"type"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	SDP       string                  `json:"sdp"`
}

type Signaling struct {
	Signal   func(signal Signal)
	OnSignal func(cb func(signal Signal))
}
type NewStreamBody struct {
	IsDirectConnect bool `json:"isDirectConnect"`
	IsPrivate       bool `json:"isPrivate"`
}

func Initialize() {
	config := utils.GetConfig()
	client := resty.New()
	client.R().SetBody(NewStreamBody{
		IsDirectConnect: config.IsDirectConnect,
		IsPrivate:       config.IsPrivate,
	}).
		Post(fmt.Sprintf("%s/connect/%s/internal", config.SignalingServer, config.StreamId))

	log.Info().Str("streamId", config.StreamId).Msg("Connected")
}

func SendSignals(outgoing_signal_chan chan Signal) {
	config := utils.GetConfig()
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
					Post(fmt.Sprintf("%s/signal/%s/internal", config.SignalingServer, config.StreamId))
			}
		}
	}
}

func PollSignals() chan Signal {
	config := utils.GetConfig()
	signalsChan := make(chan Signal, 100)
	go func() {
		client := resty.New()
		for {
			res, err := client.R().
				SetHeader("Accept", "application/json").
				SetQueryParam("isDirectConnect", fmt.Sprintf("%v", config.IsDirectConnect)).
				SetQueryParam("isPrivate", fmt.Sprintf("%v", config.IsPrivate)).
				Get(fmt.Sprintf("%s/signal/%s/internal", config.SignalingServer, config.StreamId))

			if err != nil {
				log.Err(err).Send()
				time.Sleep(time.Second * 1)
				continue
			}
			body := utils.ParseJson[[]Signal](res)
			for _, signal := range body.Value {
				log.Printf("Received signal from %s", signal.ViewerId)
				signalsChan <- signal
			}
			time.Sleep(time.Second * 1)
		}
	}()

	return signalsChan
}

func SendSnapshots() {
	config := utils.GetConfig()
	ticker := time.NewTicker(time.Second * 5)
	client := resty.New()
	sx, sy := robotgo.GetScreenSize()
	for range ticker.C {
		frame := robotgo.CaptureImg(0, 0, sx, sy)
		resized := resize.Resize(1280, 0, frame, resize.Lanczos3)
		buf := &bytes.Buffer{}
		err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: 70})
		if err != nil {
			log.Err(err).Send()
			continue
		}
		_, err = client.R().
			SetBody(buf.Bytes()).
			Post(fmt.Sprintf("%s/snapshot/%s/internal", config.SignalingServer, config.StreamId))

		if err != nil {
			log.Err(err).Send()
			continue
		}
	}
}

func NewSignaling() *Signaling {
	Initialize()
	outgoing_signal_chan := make(chan Signal, 100)
	go SendSignals(outgoing_signal_chan)
	go SendSnapshots()
	signalsChan := PollSignals()
	return &Signaling{
		Signal: func(signal Signal) { outgoing_signal_chan <- signal },
		OnSignal: func(cb func(signal Signal)) {
			go func() {
				for signal := range signalsChan {
					cb(signal)
				}
			}()
		},
	}
}
