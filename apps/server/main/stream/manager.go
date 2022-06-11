package stream

import (
	"bytes"
	"signaling/main/rtc"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/olebedev/emitter"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

type ListStreamsResponseEntry struct {
	StreamId        string `json:"streamId"`
	Viewers         int    `json:"viewers"`
	Uptime          int    `json:"uptime"`
	IsDirectConnect bool   `json:"directConnect"`
}

type Stream struct {
	Id                         string
	Connection                 *rtc.PeerConnection
	ViewerManager              *rtc.ConnectionManager
	SetSnapshot                func(snapshot *bytes.Buffer)
	GetSnapshot                func() *bytes.Buffer
	SignalToCaptureClient      func(signal rtc.Signal) error
	SignalFromCaptureClient    func(signal rtc.Signal) error
	ConnectClient              func() *rtc.PeerConnection
	NewViewer                  func(viewerId string) *rtc.PeerConnection
	GetViewer                  func(viewerId string) *rtc.PeerConnection
	GetViewerCount             func() int
	OnClientConnectionEvent    func(event ConnectionEvent)
	GetViewers                 func() map[string]*rtc.PeerConnection
	GetSignalsForCaptureClient func() chan []rtc.Signal
	IsAvailable                func() bool
	IsDirectConnect            bool
	IsPrivate                  bool
	GetUptime                  func() time.Duration
	OnViewerConnected          func(cb func(connectionId string))
	OnViewerDisconnected       func(cb func(connectionId string))
}

type StreamManager struct {
	streams               map[string]*Stream
	GetStreams            func() map[string]*Stream
	GetStream             func(streamId string) *Stream
	NewStream             func(streamId string, isDirectConnect bool, isPrivate bool) *Stream
	SetSnapshot           func(streamId string, snapshot *bytes.Buffer)
	SetP2PConnectionCount func(streamId string, count int)
	GetSnapshot           func(streamId string) *bytes.Buffer
	ListStreams           func() []ListStreamsResponseEntry
}

/////////////////////////////streamId///viewerId//signals
var to_client_signal_buffers = make(map[string][]rtc.Signal, 0)

func NewStreamManager(g *echo.Group) *StreamManager {

	//A map to store connections by their ID
	var streams = make(map[string]*Stream)
	e := &emitter.Emitter{}
	e.Use("*", emitter.Void)

	clientConnectionManager := rtc.NewConnectionManager()

	manager := &StreamManager{
		streams: streams,
		GetStreams: func() map[string]*Stream {
			return streams
		},
		GetStream: func(streamId string) *Stream {
			return streams[streamId]
		},
		NewStream: func(streamId string, isDirectConnect bool, isPrivate bool) (stream *Stream) {

			p2pConnectionCount := 0
			isAvailable := false
			keepAliveInterrupt := make(chan bool)
			uptime := time.Duration(0)
			now := time.Now()
			keepAlive := func() {

				go func() {
					select {
					case keepAliveInterrupt <- true:
					default:
					}
					select {
					case <-keepAliveInterrupt:
						return
					case <-time.After(time.Second * 15):
						isAvailable = false
					}
				}()
				uptime = time.Since(now)
				isAvailable = true
			}

			var viewer_manager *rtc.ConnectionManager

			// fix leaky subscription
			existing_stream := streams[streamId]
			if existing_stream != nil {
				viewer_manager = existing_stream.ViewerManager
			} else {
				viewer_manager = rtc.NewConnectionManager()
			}

			viewer_manager.OnAllDisconnected(func() {
				// when all viewers disconnected from this stream,
				// disconnect the server(this code) from the capture client
				clientConnectionManager.RemoveConnection(streamId)

			})

			// remove the client connection if exists(stale connection)
			clientConnectionManager.RemoveConnection(streamId)

			to_client_signal_buffers[streamId] = make([]rtc.Signal, 0)

			snapshot := bytes.NewBuffer(nil)
			stream = &Stream{
				IsDirectConnect: isDirectConnect,
				IsPrivate:       isPrivate,
				ViewerManager:   viewer_manager,
				Id:              streamId,
				Connection:      nil,
				GetUptime: func() time.Duration {
					return uptime
				},
				IsAvailable: func() bool {
					return isAvailable
				},
				GetSnapshot: func() *bytes.Buffer {
					return snapshot
				},
				SetSnapshot: func(_snapshot *bytes.Buffer) {
					keepAlive()
					snapshot = _snapshot
				},

				SignalToCaptureClient: func(signal rtc.Signal) error {

					to_client_signal_buffers[streamId] = append(to_client_signal_buffers[streamId], signal)
					return nil
				},
				SignalFromCaptureClient: func(signal rtc.Signal) error {
					keepAlive()
					return nil
				},
				GetSignalsForCaptureClient: func() chan []rtc.Signal {
					keepAlive()
					signals_to_send := make(chan []rtc.Signal)
					go func() {
						now := time.Now()
						for {

							//if 10 seconds passed, return empty array
							if time.Since(now) > 10*time.Second {
								signals_to_send <- make([]rtc.Signal, 0)
								return
							}

							if to_client_signal_buffers[streamId] != nil {
								if len(to_client_signal_buffers[streamId]) > 0 {
									signals_to_send <- (to_client_signal_buffers[streamId])

									to_client_signal_buffers[streamId] = make([]rtc.Signal, 0)
									return
								}
							}

							time.Sleep(time.Second * 1)
						}
					}()

					return signals_to_send
				},
				NewViewer: func(viewerId string) *rtc.PeerConnection {
					viewerConnection := viewer_manager.NewConnection(viewerId)

					return viewerConnection
				},
				GetViewer: func(viewerId string) *rtc.PeerConnection {
					return viewer_manager.GetConnection(viewerId)
				},
				GetViewers: func() map[string]*rtc.PeerConnection {
					return viewer_manager.GetConnections()
				},
				GetViewerCount: func() int {
					if isDirectConnect {
						return p2pConnectionCount
					}
					return len(viewer_manager.GetConnections())
				},

				OnViewerConnected: func(cb func(connection string)) {
					if isDirectConnect {
						e.On("p2p_viewer_connected", func(ev *emitter.Event) {
							go cb(ev.Args[0].(string))
						})
					} else {
						viewer_manager.OnConnection(cb)
					}
				},
				OnViewerDisconnected: func(cb func(connectionId string)) {
					if isDirectConnect {
						e.On("p2p_viewer_disconnected", func(ev *emitter.Event) {
							go cb(ev.Args[0].(string))
						})
					} else {
						viewer_manager.OnDisconnected(cb)
					}
				},
				OnClientConnectionEvent: func(event ConnectionEvent) {
					if !isDirectConnect {
						return
					}
					p2pConnectionCount = event.ViewerCount
					if event.Type == "viewer_connected" {
						go e.Emit("p2p_viewer_connected", event.ViewerId)
					} else {
						go e.Emit("p2p_viewer_disconnected", event.ViewerId)
					}
				},
				ConnectClient: func() *rtc.PeerConnection {
					// local peer connection
					conn := clientConnectionManager.GetConnection(streamId)

					if conn == nil {
						conn = clientConnectionManager.NewConnection(streamId)
					}

					go func() {
						// 30 fps ticker
						ticker := time.NewTicker(time.Second / 30)
						for range ticker.C {
							if conn.ConnectionState() == webrtc.PeerConnectionStateClosed {
								log.Error().
									Str("streamId", streamId).
									Msg("client disconnected")
								// connection to the capture client, set to nil, so that it can be recreated
								stream.Connection = nil
								return
							}
						}
					}()

					conn.OnSignal(func(signal rtc.Signal) {

						// forward the signal to the capture client
						to_client_signal_buffers[streamId] = append(
							to_client_signal_buffers[streamId],
							signal)
					})

					// allow receiving tracks from the capture client
					conn.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
					conn.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
					dc, err := conn.CreateDataChannel("data", nil)
					if err != nil {
						log.Error().
							Str("streamId", streamId).
							Msg("failed to create data channel")
					}

					conn.DataChannel = dc

					// initiate the peer connection with an offer to the capture client
					conn.Initiate()
					stream.Connection = conn
					return stream.Connection
				},
			}

			streams[streamId] = stream
			return stream
		},
		SetSnapshot: func(streamId string, snapshot *bytes.Buffer) {
			streams[streamId].SetSnapshot(snapshot)
		},

		GetSnapshot: func(streamId string) *bytes.Buffer {
			return streams[streamId].GetSnapshot()
		},
		ListStreams: func() []ListStreamsResponseEntry {
			response := make([]ListStreamsResponseEntry, 0)

			for streamId_runId, stream := range streams {
				if !stream.IsAvailable() || stream.IsPrivate {
					continue
				}
				streamId := streamId_runId[:len(streamId_runId)-len(runId)]
				response = append(response, ListStreamsResponseEntry{
					StreamId:        streamId,
					Viewers:         stream.GetViewerCount(),
					Uptime:          int(stream.GetUptime().Seconds()),
					IsDirectConnect: stream.IsDirectConnect,
				})
			}
			return response
		},
	}

	return manager

}
