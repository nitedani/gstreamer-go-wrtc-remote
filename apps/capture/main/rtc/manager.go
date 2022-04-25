package rtc

import (
	"time"

	"github.com/olebedev/emitter"
	"github.com/pion/webrtc/v3"
)

type ConnectionManager struct {
	connections       map[string]*PeerConnection
	GetConnections    func() map[string]*PeerConnection
	GetConnection     func(viewerId string) *PeerConnection
	NewConnection     func(viewerId string) *PeerConnection
	OnAllDisconnected func(cb func())
	OnFirstConnection func(cb func())
}

func NewConnectionManager() *ConnectionManager {
	//A map to store connections by their ID
	var connections = make(map[string]*PeerConnection)
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)
	numConnections := 0

	manager := &ConnectionManager{
		connections: connections,
		GetConnections: func() map[string]*PeerConnection {
			return connections
		},
		GetConnection: func(viewerId string) *PeerConnection {
			return connections[viewerId]
		},
		NewConnection: func(viewerId string) *PeerConnection {
			connection := newConnection(viewerId)
			connections[connection.ViewerId] = connection

			connection.OnConnected(func() {
				numConnections++
				if numConnections == 1 {
					e.Emit("firstconnection")
				}
			})

			connection.OnDisconnected(func() {
				numConnections--
				delete(connections, connection.ViewerId)
				if numConnections == 0 {
					e.Emit("alldisconnected")
				}
			})
			return connection
		},
		OnAllDisconnected: func(cb func()) {
			e.On("alldisconnected", func(e *emitter.Event) {
				cb()
			})
		},
		OnFirstConnection: func(cb func()) {
			e.On("firstconnection", func(e *emitter.Event) {
				cb()
			})
		},
	}

	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for range ticker.C {
			all_disconnect := true
			for _, connection := range connections {
				if connection.ConnectionState() != webrtc.PeerConnectionStateClosed {
					all_disconnect = false
				} else {
					delete(connections, connection.ViewerId)
				}
			}
			if all_disconnect {
				numConnections = 0
				e.Emit("alldisconnected")
			}
		}
	}()

	return manager
}
