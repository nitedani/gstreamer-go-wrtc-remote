package rtc

import (
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type RtcConfig struct {
	VideoMimeType string
	AudioMimeType string
}

func GetRtcConfig() RtcConfig {

	encoder, hasEnv := os.LookupEnv("ENCODER")
	if !hasEnv {
		log.Info().Msg("No encoder specified, defaulting to vp8")
		encoder = "vp8"
	}

	var videoMimeType string

	switch encoder {
	case "vp8":
		videoMimeType = webrtc.MimeTypeVP8
	case "h264":
		videoMimeType = webrtc.MimeTypeH264
	case "nvenc":
		videoMimeType = webrtc.MimeTypeH264
	default:
		log.Fatal().Msg("Invalid encoder specified")
	}

	audioMimeType := webrtc.MimeTypeOpus

	return RtcConfig{

		VideoMimeType: videoMimeType,
		AudioMimeType: audioMimeType,
	}
}
