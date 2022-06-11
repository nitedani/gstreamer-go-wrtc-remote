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

				go func() {
					e.Emit("connection", connection.Id)
				}()

				/*
					if numConnections == 1 {
						go func() {
							e.Emit("firstconnection")
						}()
					}
				*/
			})

			connection.OnDisconnected(func() {
				numConnections--
				delete(connections, connection.Id)

				go func() {
					e.Emit("disconnected", connection.Id)
				}()

				if numConnections == 0 {
					go func() {
						e.Emit("alldisconnected")
					}()
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
			go func() {
				for range e.On("alldisconnected") {
					go cb()
				}
			}()
		},
		OnFirstConnection: func(cb func()) {
			go func() {
				for range e.On("firstconnection") {
					go cb()
				}
			}()
		},
		OnConnection: func(cb func(connectionId string)) {
			go func() {
				for ev := range e.On("connection") {
					cb(ev.Args[0].(string))
				}
			}()
		},
		OnDisconnected: func(cb func(connectionId string)) {
			go func() {
				for ev := range e.On("disconnected") {
					cb(ev.Args[0].(string))
				}
			}()
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
