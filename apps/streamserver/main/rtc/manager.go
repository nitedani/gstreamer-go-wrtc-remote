package rtc

import "github.com/olebedev/emitter"

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

	return &ConnectionManager{
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
			length := len(connections)

			connection.OnConnected(func() {
				if length == 1 {
					e.Emit("firstconnection")
				}
			})

			connection.OnDisconnected(func() {
				delete(connections, connection.ViewerId)
				if len(connections) == 0 {
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
}
