package rtc

import (
	"github.com/olebedev/emitter"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type ConnectionManager struct {
	connections       map[string]*PeerConnection
	GetConnections    func() map[string]*PeerConnection
	GetConnection     func(connectionId string) *PeerConnection
	NewConnection     func(connectionId string) *PeerConnection
	RemoveConnection  func(connectionId string)
	OnAllDisconnected func(cb func())
	OnFirstConnection func(cb func())
	OnConnection      func(cb func(connectionId string))
	OnDisconnected    func(cb func(connectionId string))
	*emitter.Emitter
}

func NewConnectionManager() *ConnectionManager {
	//A map to store connections by their ID
	var connections = make(map[string]*PeerConnection)
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)
	numConnections := 0

	manager := &ConnectionManager{
		Emitter:     e,
		connections: connections,
		GetConnections: func() map[string]*PeerConnection {
			return connections
		},
		GetConnection: func(connectionId string) *PeerConnection {
			return connections[connectionId]
		},
		NewConnection: func(connectionId string) *PeerConnection {
			log.Error().Str("connectionId", connectionId).Msg("NewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnectionNewConnection")
			connection := newConnection(connectionId)

			connections[connection.Id] = connection

			connection.OnConnected(func() {
				numConnections++
				e.Emit("connection", connection.Id)
				if numConnections == 1 {
					e.Emit("firstconnection")
				}
			})

			connection.OnDisconnected(func() {
				numConnections--
				delete(connections, connection.Id)
				e.Emit("disconnected", connection.Id)
				if numConnections == 0 {
					e.Emit("alldisconnected")
				}
				// e.Off("*")
			})

			connection.OnDataChannel(func(dc *webrtc.DataChannel) {
				connection.DataChannel = dc
			})

			return connection
		},
		RemoveConnection: func(connectionId string) {
			connection := connections[connectionId]
			if connection != nil {

				connection.EmitterVoid.Emit("disconnected")

				connection.Close()
				delete(connections, connectionId)
			}
		},
		OnAllDisconnected: func(cb func()) {
			e.On("alldisconnected", func(e *emitter.Event) {
				go cb()
			})
		},
		OnFirstConnection: func(cb func()) {
			e.On("firstconnection", func(e *emitter.Event) {
				go cb()
			})
		},
		OnConnection: func(cb func(connectionId string)) {
			e.On("connection", func(ev *emitter.Event) {
				go cb(ev.Args[0].(string))
			})
		},
		OnDisconnected: func(cb func(connectionId string)) {
			e.On("disconnected", func(ev *emitter.Event) {
				go cb(ev.Args[0].(string))
			})
		},
	}
	/*
		go func() {
			ticker := time.NewTicker(time.Second * 5)
			for range ticker.C {
				for _, connection := range connections {
					if connection.ConnectionState() == webrtc.PeerConnectionStateClosed {
						numConnections--
						delete(connections, connection.Id)
						if numConnections == 0 {
							e.Emit("alldisconnected")
						}
					}
				}
			}
		}()
	*/
	return manager

}
