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

	videoCapture := capture.NewVideoCapture()
	audioCapture := capture.NewAudioCapture()

	trackWriter := rtc.NewTrackWriter(videoCapture, audioCapture)

	connectionManager.OnFirstConnection(func() {
		trackWriter.Start()
	})

	connectionManager.OnAllDisconnected(func() {
		trackWriter.Stop()
	})

	signaling.OnSignal(func(signal rtc.Signal) {
		viewerId := signal.ViewerId
		connection := connectionManager.GetConnection(viewerId)

		if connection == nil {

			connection = connectionManager.NewConnection(viewerId)

			connection.AddTracks(trackWriter.Tracks)

			connection.OnSignal(func(_signal rtc.Signal) {
				signaling.Signal(_signal)
			})

			connection.OnConnected(func() {
				log.Info().Str("viewerId", viewerId).Msg("Connected")
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
