package rtc

import (
	"server/main/capture"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog/log"
)

type Tracks struct {
	VideoTrack *webrtc.TrackLocalStaticSample
	AudioTrack *webrtc.TrackLocalStaticSample
}

type SetupTracksReturnType struct {
	Tracks *Tracks
	Start  func()
	Stop   func()
}

func NewTrackWriter(videoCapture *capture.ControlledCapture, audioCapture *capture.ControlledCapture) SetupTracksReturnType {

	config := GetRtcConfig()

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: config.VideoMimeType, ClockRate: 90000}, "video", "pion")
	if err != nil {
		panic(err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: config.AudioMimeType, ClockRate: 48000}, "audio", "pion")
	if err != nil {
		panic(err)
	}

	stopped := true

	sendVideo := func() {

		videoSubscription, videoCleanup := videoCapture.GetChannel()
		for frame_buffer := range videoSubscription {
			if stopped {
				videoCleanup()
				return
			}

			err = videoTrack.WriteSample(media.Sample{Data: frame_buffer.Bytes(), Duration: frame_buffer.Duration()})
			if err != nil {
				log.Err(err).Send()
				videoCleanup()
				return
			}
		}
	}

	sendAudio := func() {
		audioSubscription, audioCleanup := audioCapture.GetChannel()
		for sample_buffer := range audioSubscription {
			if stopped {
				audioCleanup()
				return
			}

			err = audioTrack.WriteSample(media.Sample{Data: sample_buffer.Bytes(), Duration: sample_buffer.Duration()})
			if err != nil {
				log.Err(err).Send()
				audioCleanup()
				return
			}
		}
	}

	start := func() {
		if stopped {
			stopped = false
			go sendVideo()
			go sendAudio()
		}
	}

	stop := func() {
		stopped = true
	}

	return SetupTracksReturnType{
		Tracks: &Tracks{
			VideoTrack: videoTrack,
			AudioTrack: audioTrack,
		},
		Start: start,
		Stop:  stop,
	}

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
