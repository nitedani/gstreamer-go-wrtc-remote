package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

/*
{
  "settings": {
    "stream_id": "default",
    "remote_enabled": false,
    "direct_connect": true,
    "bitrate": 15388600,
    "resolution": "1920x1080",
    "framerate": 90,
    "encoder": "nvenc",
    "threads": 4,
    "server_url": "http://localhost:4000/api"
  }
}

*/

type ConfigFileSettings struct {
	StreamId        string `json:"stream_id"`
	RemoteEnabled   bool   `json:"remote_enabled"`
	IsDirectConnect bool   `json:"direct_connect"`
	IsPrivate       bool   `json:"private"`
	Bitrate         int    `json:"bitrate"`
	Resolution      string `json:"resolution"`
	Framerate       int    `json:"framerate"`
	Encoder         string `json:"encoder"`
	Threads         int    `json:"threads"`
	SignalingServer string `json:"server_url"`
}
type ConfigFile struct {
	Settigs ConfigFileSettings `json:"settings"`
}

type Config struct {
	RemoteEnabled   bool
	IsDirectConnect bool
	IsPrivate       bool
	SignalingServer string
	StreamId        string
	Bitrate         int
	Resolution      string
	ResolutionX     string
	ResolutionY     string
	Framerate       int
	Threads         int
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

	configPath := os.Args[1]

	jsonFile, err := os.Open(configPath)
	if err != nil {
		log.Fatal().Msgf("Error opening config file: %s", err)
	}
	var parsedConfig ConfigFile
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal().Msgf("Error reading config file: %s", err)
	}

	err = json.Unmarshal(byteValue, &parsedConfig)
	if err != nil {
		log.Fatal().Msgf("Error parsing config file: %s", err)
	}
	settings := parsedConfig.Settigs
	sizes := strings.Split(settings.Resolution, "x")

	config = &Config{
		RemoteEnabled:   settings.RemoteEnabled,
		IsDirectConnect: settings.IsDirectConnect,
		IsPrivate:       settings.IsPrivate,
		SignalingServer: settings.SignalingServer,
		StreamId:        settings.StreamId,
		Bitrate:         settings.Bitrate,
		Resolution:      settings.Resolution,
		ResolutionX:     sizes[0],
		ResolutionY:     sizes[1],
		Framerate:       settings.Framerate,
		Threads:         settings.Threads,
		Encoder:         settings.Encoder,
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
