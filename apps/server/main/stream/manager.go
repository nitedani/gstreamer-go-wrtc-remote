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
	StreamId string `json:"streamId"`
	Viewers  int    `json:"viewers"`
	Uptime   int    `json:"uptime"`
}

type Stream struct {
	Id                         string
	Connection                 *rtc.PeerConnection
	SetSnapshot                func(snapshot *bytes.Buffer)
	GetSnapshot                func() *bytes.Buffer
	SignalToCaptureClient      func(signal rtc.Signal) error
	SignalFromCaptureClient    func(signal rtc.Signal) error
	ConnectClient              func() *rtc.PeerConnection
	NewViewer                  func(viewerId string) *rtc.PeerConnection
	GetViewer                  func(viewerId string) *rtc.PeerConnection
	GetViewerCount             func() int
	GetViewers                 func() map[string]*rtc.PeerConnection
	GetSignalsForCaptureClient func() chan []rtc.Signal
	GetSignalsForViewer        func(viewerId string) chan []rtc.Signal
	IsAvailable                func() bool
	GetUptime                  func() time.Duration
}

type StreamManager struct {
	streams     map[string]*Stream
	GetStreams  func() map[string]*Stream
	GetStream   func(streamId string) *Stream
	NewStream   func(streamId string) *Stream
	SetSnapshot func(streamId string, snapshot *bytes.Buffer)
	GetSnapshot func(streamId string) *bytes.Buffer
	ListStreams func() []ListStreamsResponseEntry
}

/////////////////////////////streamId///viewerId//signals
var to_client_signal_buffers = make(map[string][]rtc.Signal, 0)

/////////////////////////////streamId///viewerId//signals
var to_viewer_signal_buffers = make(map[string]map[string][]rtc.Signal, 0)

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
		NewStream: func(streamId string) (stream *Stream) {

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
					case <-time.After(time.Second * 6):
						isAvailable = false
					}
				}()
				uptime = time.Since(now)
				isAvailable = true
			}

			viewer_manager := rtc.NewConnectionManager()
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
				Id:         streamId,
				Connection: nil,
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
					viewerId := signal.ViewerId
					if to_viewer_signal_buffers[streamId] == nil {
						to_viewer_signal_buffers[streamId] = make(map[string][]rtc.Signal, 0)
					}

					if to_viewer_signal_buffers[streamId][viewerId] == nil {
						to_viewer_signal_buffers[streamId][viewerId] = make([]rtc.Signal, 0)
					}

					to_viewer_signal_buffers[streamId][viewerId] =
						append(to_viewer_signal_buffers[streamId][viewerId], signal)
					return nil
				},
				GetSignalsForCaptureClient: func() chan []rtc.Signal {
					keepAlive()
					signals_to_send := make(chan []rtc.Signal)
					go func() {
						now := time.Now()
						for {

							//if 20 seconds passed, return empty array
							if time.Since(now) > 20*time.Second {
								signals_to_send <- make([]rtc.Signal, 0)
								return
							}

							if len(to_client_signal_buffers[streamId]) > 0 {
								signals_to_send <- (to_client_signal_buffers[streamId])
								to_client_signal_buffers[streamId] = make([]rtc.Signal, 0)
								return
							}

							time.Sleep(time.Second * 1)
						}
					}()

					return signals_to_send
				},
				GetSignalsForViewer: func(viewerId string) chan []rtc.Signal {
					signals_to_send := make(chan []rtc.Signal)
					go func() {
						now := time.Now()
						for {
							//if 20 seconds passed, return empty array
							if time.Since(now) > 20*time.Second {
								signals_to_send <- make([]rtc.Signal, 0)
								return
							}

							if to_viewer_signal_buffers[streamId] == nil {
								to_viewer_signal_buffers[streamId] = make(map[string][]rtc.Signal, 0)
							}

							if to_viewer_signal_buffers[streamId][viewerId] == nil {
								to_viewer_signal_buffers[streamId][viewerId] = make([]rtc.Signal, 0)
							}

							// wait until signal_buffer[id] is not empty
							if len(to_viewer_signal_buffers[streamId][viewerId]) > 0 {
								signals_to_send <- (to_viewer_signal_buffers[streamId][viewerId])
								to_viewer_signal_buffers[streamId][viewerId] = make([]rtc.Signal, 0)
								return
							}
							time.Sleep(time.Second * 1)
						}
					}()
					return signals_to_send
				},
				NewViewer: func(viewerId string) *rtc.PeerConnection {
					viewerConnection := viewer_manager.NewConnection(viewerId)
					viewerConnection.OnSignal(func(signal rtc.Signal) {
						if to_viewer_signal_buffers[streamId] == nil {
							to_viewer_signal_buffers[streamId] = make(map[string][]rtc.Signal, 0)
						}

						if to_viewer_signal_buffers[streamId][viewerId] == nil {
							to_viewer_signal_buffers[streamId][viewerId] = make([]rtc.Signal, 0)
						}
						to_viewer_signal_buffers[streamId][viewerId] =
							append(to_viewer_signal_buffers[streamId][viewerId], signal)
					})
					return viewerConnection
				},
				GetViewer: func(viewerId string) *rtc.PeerConnection {
					return viewer_manager.GetConnection(viewerId)
				},
				GetViewers: func() map[string]*rtc.PeerConnection {
					return viewer_manager.GetConnections()
				},
				GetViewerCount: func() int {
					return len(viewer_manager.GetConnections())
				},
				ConnectClient: func() *rtc.PeerConnection {
					conn := clientConnectionManager.NewConnection(streamId)

					conn.OnDisconnected(func() {
						log.Error().
							Str("streamId", streamId).
							Msg("client disconnected")
						// when the capture client disconnects, remove the stream, new viewers will be rejected
						delete(to_client_signal_buffers, streamId)
						stream.Connection = nil
					})

					conn.OnSignal(func(signal rtc.Signal) {
						// forward the signal to the capture client
						to_client_signal_buffers[streamId] = append(
							to_client_signal_buffers[streamId],
							signal)
					})

					// allow receiving tracks from the capture client
					conn.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
					conn.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})

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
				if !stream.IsAvailable() {
					continue
				}
				streamId := streamId_runId[:len(streamId_runId)-len(runId)]
				response = append(response, ListStreamsResponseEntry{
					StreamId: streamId,
					Viewers:  stream.GetViewerCount(),
					Uptime:   int(stream.GetUptime().Seconds()),
				})
			}
			return response
		},
	}

	return manager

}
