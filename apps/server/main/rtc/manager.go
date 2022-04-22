package rtc

import (
	"github.com/olebedev/emitter"
)

type ConnectionManager struct {
	connections       map[string]*PeerConnection
	GetConnections    func() map[string]*PeerConnection
	GetConnection     func(connectionId string) *PeerConnection
	NewConnection     func(connectionId string) *PeerConnection
	RemoveConnection  func(connectionId string)
	OnAllDisconnected func(cb func())
	OnFirstConnection func(cb func())
}

func NewConnectionManager() *ConnectionManager {
	//A map to store connections by their ID
	var connections = make(map[string]*PeerConnection)
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)
	numConnections := 0

	return &ConnectionManager{
		connections: connections,
		GetConnections: func() map[string]*PeerConnection {
			return connections
		},
		GetConnection: func(connectionId string) *PeerConnection {
			return connections[connectionId]
		},
		NewConnection: func(connectionId string) *PeerConnection {
			connection := newConnection(connectionId)
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
		RemoveConnection: func(connectionId string) {
			connections[connectionId].Close()
			delete(connections, connectionId)
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
