package rtc

import (
	"client/utils"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type RtcConfig struct {
	VideoMimeType string
	AudioMimeType string
}

func GetRtcConfig() RtcConfig {
	config := utils.GetConfig()

	var videoMimeType string
	switch config.Encoder {
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
