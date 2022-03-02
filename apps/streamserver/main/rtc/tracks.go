package rtc

import (
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
)

type StreamTracks struct {
	VideoTrack *webrtc.TrackLocalStaticSample
	AudioTrack *webrtc.TrackLocalStaticSample
}

type SetupTracksReturnType struct {
	StreamTracks *StreamTracks
	Start        func()
	Stop         func()
}

func SetupTracks(getVideoChannelFn func() chan *gst.Buffer, getAudioChannelFn func() chan *gst.Buffer) SetupTracksReturnType {
	encoder, hasEnv := os.LookupEnv("ENCODER")
	if !hasEnv {
		encoder = "vp8"
	}

	var videoEncoder string

	switch encoder {
	case "vp8":
		videoEncoder = webrtc.MimeTypeVP8
	case "vp9":
		videoEncoder = webrtc.MimeTypeVP9
	case "h264":
	case "nvenc":
		videoEncoder = webrtc.MimeTypeH264
	default:
		panic("Invalid encoder")
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: videoEncoder, ClockRate: 90000}, "video", "pion")
	if err != nil {
		panic(err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}, "audio", "pion")
	if err != nil {
		panic(err)
	}

	stopped := true
	sendVideo := func() {
		channel := getVideoChannelFn()
		for frame_buffer := range channel {
			if stopped {
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
			if stopped {
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
		StreamTracks: &StreamTracks{
			VideoTrack: videoTrack,
			AudioTrack: audioTrack,
		},
		Start: start,
		Stop:  stop,
	}

}

func AttachTracks(peerConnection *PeerConnection, tracks *StreamTracks) {
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
