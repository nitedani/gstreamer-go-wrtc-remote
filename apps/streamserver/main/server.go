package main

import (
	"server/main/capture"
	"server/main/remote"
	"server/main/rtc"
	"server/main/utils"

	"github.com/rs/zerolog/log"
)

func StartWrtcServer() {

	config := utils.GetConfig()
	signaling := rtc.NewSignaling()
	connectionManager := rtc.NewConnectionManager()

	videoCapture := capture.CreateVideoCapture()
	audioCapture := capture.CreateAudioCapture()

	tracks := rtc.SetupTracks(videoCapture, audioCapture)

	connectionManager.OnFirstConnection(func() {
		tracks.Start()
	})

	connectionManager.OnAllDisconnected(func() {
		tracks.Stop()
	})

	signaling.OnSignal(func(signal rtc.Signal) {
		viewerId := signal.ViewerId
		connection := connectionManager.GetConnection(viewerId)

		if connection == nil {

			log.Info().Str("viewerId", viewerId).Msg("Connected")

			connection = connectionManager.NewConnection(viewerId)
			connection.AttachTracks(tracks.StreamTracks)
			connection.OnSignal(func(_signal rtc.Signal) {
				signaling.Signal(_signal)
			})

			connection.OnDisconnected(func() {
				log.Info().Str("viewerId", viewerId).Msg("Disconnected")
			})

			if config.RemoteEnabled {
				remote.SetupRemoteControl(connection)
			}

		}

		connection.Signal(signal)
	})
}
