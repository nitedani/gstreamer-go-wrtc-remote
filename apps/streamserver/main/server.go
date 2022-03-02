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

	connectionManager := rtc.NewConnectionManager()
	signaling := rtc.NewSignaling()
	tracks := rtc.SetupTracks(capture.CreateVideoCapture(), capture.CreateAudioCapture())

	connectionManager.OnAllDisconnected(func() {
		tracks.Stop()
	})

	connectionManager.OnFirstConnection(func() {
		tracks.Start()
	})

	signaling.OnSignal(func(signal rtc.Signal) {
		viewerId := signal.ViewerId
		connection := connectionManager.GetConnection(viewerId)

		if connection == nil {

			log.Info().Str("viewerId", viewerId).Msg("Connected")

			connection = connectionManager.NewConnection(viewerId)
			rtc.AttachTracks(connection, tracks.StreamTracks)
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
