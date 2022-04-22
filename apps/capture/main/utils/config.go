package utils

import (
	"os"
	"strconv"
	"strings"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type Config struct {
	RemoteEnabled   bool
	SignalingServer string
	StreamId        string
	Bitrate         string
	Resolution      string
	ResolutionX     string
	ResolutionY     string
	Framerate       string
	Threads         string
	Encoder         string
}

type MediaConfig struct {
	Config
	VideoPipeline string
	AudioPipeline string
	VideoMimeType string
	AudioMimeType string
}

var config *Config

func initConfig() {
	remoteEnabled, hasEnv := os.LookupEnv("REMOTE_ENABLED")
	if !hasEnv {
		remoteEnabled = "false"
	}

	remoteEnabledBool, err := strconv.ParseBool(remoteEnabled)
	if err != nil {
		log.Err(err).Msg("Failed to parse REMOTE_ENABLED")
		remoteEnabledBool = false
	}

	signalingServer, hasEnv := os.LookupEnv("SERVER_URL")
	if !hasEnv {
		panic("SERVER_URL not set")
	}

	streamId, hasEnv := os.LookupEnv("STREAM_ID")
	if !hasEnv {
		panic("STREAM_ID not set")
	}

	bitrate, hasEnv := os.LookupEnv("BITRATE")
	if !hasEnv {
		log.Info().Msg("No bitrate specified, defaulting to 5Mbps")
		bitrate = "5242880"
	}

	resolution, hasEnv := os.LookupEnv("RESOLUTION")
	if !hasEnv {
		log.Info().Msg("No resolution specified, defaulting to 1280x720")
		resolution = "1280x720"
	}
	sizes := strings.Split(resolution, "x")

	framerate, hasEnv := os.LookupEnv("FRAMERATE")
	if !hasEnv {
		log.Info().Msg("No framerate specified, defaulting to 30fps")
		framerate = "30"
	}

	threads, hasEnv := os.LookupEnv("THREADS")
	if !hasEnv {
		log.Info().Msg("No threads specified, defaulting to 2")
		threads = "2"
	}

	encoder, hasEnv := os.LookupEnv("ENCODER")
	if !hasEnv {
		log.Info().Msg("No encoder specified, defaulting to vp8")
		encoder = "vp8"
	}

	config = &Config{
		RemoteEnabled:   remoteEnabledBool,
		SignalingServer: signalingServer,
		StreamId:        streamId,
		Bitrate:         bitrate,
		Resolution:      resolution,
		ResolutionX:     sizes[0],
		ResolutionY:     sizes[1],
		Framerate:       framerate,
		Threads:         threads,
		Encoder:         encoder,
	}

}

func GetConfig() Config {
	if config == nil {
		initConfig()
	}
	return *config
}

func GetMediaConfig() MediaConfig {

	config := GetConfig()
	var videoPipeline string
	var videoMimeType string

	switch config.Encoder {
	case "vp8":
		videoPipeline = WinVP8Pipeline()
		videoMimeType = webrtc.MimeTypeVP8
	case "h264":
		videoPipeline = WinOpenH264Pipeline()
		videoMimeType = webrtc.MimeTypeH264
	case "nvenc":
		videoPipeline = WinNvH264Pipeline()
		videoMimeType = webrtc.MimeTypeH264
	default:
		log.Fatal().Msg("Invalid encoder specified")
	}

	audioPipeline := WinOpusPipeline()
	audioMimeType := webrtc.MimeTypeOpus

	return MediaConfig{
		Config:        config,
		VideoPipeline: videoPipeline,
		AudioPipeline: audioPipeline,
		VideoMimeType: videoMimeType,
		AudioMimeType: audioMimeType,
	}
}
