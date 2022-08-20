package main

import (
	"client/capture"
	"client/remote"
	"client/rtc"

	"github.com/rs/zerolog/log"
)

func StartWrtcServer() {

	connectionManager := rtc.NewConnectionManager()

	signaling := rtc.NewSignaling(connectionManager)

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

			go remote.SetupRemote(connection)

		}

		connection.Signal(signal)
	})
}
