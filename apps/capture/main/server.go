package main

import (
	"capture/main/capture"
	"capture/main/remote"
	"capture/main/rtc"

	"github.com/rs/zerolog/log"
)

func StartWrtcServer() {

	signaling := rtc.NewSignaling()

	videoCapture := capture.NewVideoCapture()
	audioCapture := capture.NewAudioCapture()

	connectionManager := rtc.NewConnectionManager()
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

			go remote.SetupRemote(connection)

		}

		connection.Signal(signal)
	})
}
