package rtc

import (
	"bytes"
	"time"

	"github.com/olebedev/emitter"
	"github.com/pion/webrtc/v3"
)

type ConnectionManager struct {
	connections       map[string]*PeerConnection
	GetConnections    func() map[string]*PeerConnection
	GetConnection     func(connectionId string) *PeerConnection
	NewConnection     func(connectionId string) *PeerConnection
	RemoveConnection  func(connectionId string)
	SetSnapshot       func(connectionId string, snapshot *bytes.Buffer)
	GetSnapshot       func(connectionId string) *bytes.Buffer
	OnAllDisconnected func(cb func())
	OnFirstConnection func(cb func())
}

func NewConnectionManager() *ConnectionManager {
	//A map to store connections by their ID
	var connections = make(map[string]*PeerConnection)
	var snapshots = make(map[string]*bytes.Buffer)
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)
	numConnections := 0

	manager := &ConnectionManager{
		connections: connections,
		GetConnections: func() map[string]*PeerConnection {
			return connections
		},
		GetConnection: func(connectionId string) *PeerConnection {
			return connections[connectionId]
		},
		NewConnection: func(connectionId string) *PeerConnection {
			connection := newConnection(connectionId)
			connections[connection.Id] = connection

			connection.OnConnected(func() {
				numConnections++
				if numConnections == 1 {
					e.Emit("firstconnection")
				}
			})

			connection.OnDisconnected(func() {
				numConnections--
				delete(connections, connection.Id)
				if numConnections == 0 {
					e.Emit("alldisconnected")
				}
			})

			return connection
		},
		RemoveConnection: func(connectionId string) {
			if connections[connectionId] != nil {
				connections[connectionId].Close()
				delete(connections, connectionId)
			}
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
		SetSnapshot: func(connectionId string, snapshot *bytes.Buffer) {
			snapshots[connectionId] = snapshot
		},
		GetSnapshot: func(connectionId string) *bytes.Buffer {
			return snapshots[connectionId]
		},
	}

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

	return manager

}
