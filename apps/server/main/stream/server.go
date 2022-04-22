package stream

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"signaling/main/rtc"
	"signaling/main/utils"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

func randomStr() string {
	n := 5
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}

var runId = randomStr()

/////////////////////////////streamId///viewerId//signals
var to_server_signal_buffers = make(map[string][]rtc.Signal, 0)

/////////////////////////////streamId///viewerId//signals
var to_viewer_signal_buffers = make(map[string]map[string][]rtc.Signal, 0)

/////////////////////////////streamId
var stream_managers = make(map[string]*rtc.ConnectionManager, 0)

func getViewerId(c echo.Context) string {
	id_cookie, err := c.Cookie("connection_id")
	if err != nil {
		panic(err)
	}
	return id_cookie.Value
}

func StartSignalingServer(g *echo.Group) {
	config := rtc.GetRtcConfig()
	iceServers := config.ICEServers
	directConnect := config.DirectConnect

	clientConnectionManager := rtc.NewConnectionManager()

	g.GET("/ice-config", func(c echo.Context) error {

		log.Info().
			Msg("client called /ice-config")

		return c.JSON(http.StatusOK, iceServers)
	})

	// Client route
	g.POST("/connect", func(c echo.Context) error {

		viewerId := randomStr()
		log.Info().
			Str("viewerId", viewerId).
			Msg("client called /connect")

		c.SetCookie(&http.Cookie{
			Name:  "connection_id",
			Value: viewerId,
		})
		return c.String(http.StatusOK, viewerId)
	})

	// Client route
	g.GET("/signal/:streamId", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		viewerId := getViewerId(c)
		signals_to_send := make(chan []rtc.Signal, 0)

		log.Info().
			Str("method", "GET").
			Str("viewerId", viewerId).
			Str("streamId", streamId).
			Msg("client called /signal/:streamId")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
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

		json, _ := json.Marshal(<-signals_to_send)
		return c.String(http.StatusOK, string(json))
	})

	// Client route
	g.POST("/signal/:streamId", func(c echo.Context) error {

		streamId := c.PathParam("streamId")
		viewerId := getViewerId(c)

		log.Info().
			Str("method", "POST").
			Str("viewerId", viewerId).
			Str("streamId", streamId).
			Msg("client called /signal/:streamId")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId

		// if to_server_signal_buffers[streamId] == nil, that means the client is not connected to the server
		if to_server_signal_buffers[streamId] == nil {
			return c.String(http.StatusNotFound, "stream not found")
		}

		signal := utils.ParseBody[rtc.Signal](c)
		signal.Value.ViewerId = viewerId

		if directConnect {
			to_server_signal_buffers[streamId] = append(
				to_server_signal_buffers[streamId],
				signal.Value)
		} else {

			// One per stream client
			sc := clientConnectionManager.GetConnection(streamId)

			// build the sc between the server and client, if not exist
			if sc == nil {
				sc = clientConnectionManager.NewConnection(streamId)
				sc.OnSignal(func(signal rtc.Signal) {
					to_server_signal_buffers[streamId] = append(
						to_server_signal_buffers[streamId],
						signal)
				})

				sc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
				sc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})

				sc.Initiate()
			}

			if stream_managers[streamId] == nil {
				stream_managers[streamId] = rtc.NewConnectionManager()
				stream_managers[streamId].OnAllDisconnected(func() {
					// when all viewers disconnected from this stream,
					// disconnect the server(this code) from the capture client
					clientConnectionManager.RemoveConnection(streamId)
				})
			}

			vc := stream_managers[streamId].GetConnection(viewerId)

			// build the vc between the browser and server
			if vc == nil {
				vc = stream_managers[streamId].NewConnection(viewerId)
				vc.OnSignal(func(signal rtc.Signal) {
					if to_viewer_signal_buffers[streamId] == nil {
						to_viewer_signal_buffers[streamId] = make(map[string][]rtc.Signal, 0)
					}

					if to_viewer_signal_buffers[streamId][viewerId] == nil {
						to_viewer_signal_buffers[streamId][viewerId] = make([]rtc.Signal, 0)
					}
					to_viewer_signal_buffers[streamId][viewerId] =
						append(to_viewer_signal_buffers[streamId][viewerId], signal)
				})
				// forward the tracks received from the capture client to the viewer
				sc.ConnectTo(vc)
			}

			vc.Signal(signal.Value)

		}

		return c.String(http.StatusOK, "OK")
	})

	// Server route
	g.GET("/signal/:streamId/internal", func(c echo.Context) error {

		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Str("method", "GET").
					Str("streamId", c.PathParam("streamId")).
					Str("viewerId", c.QueryParam("viewerId")).
					Str("error", fmt.Sprint(r)).
					Msg("server called /signal/:streamId/internal")
			}
		}()

		streamId := c.PathParam("streamId")
		signals_to_send := make(chan []rtc.Signal, 0)

		log.Info().
			Str("method", "GET").
			Str("streamId", streamId).
			Msg("server called /signal/:streamId/internal")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId

		if to_server_signal_buffers[streamId] == nil {
			to_server_signal_buffers[streamId] = make([]rtc.Signal, 0)
		}

		go func() {
			now := time.Now()
			for {

				select {
				case <-c.Request().Context().Done():
					return
				default:
				}
				//if 20 seconds passed, return empty array
				if time.Since(now) > 20*time.Second {
					signals_to_send <- make([]rtc.Signal, 0)
					return
				}

				// wait until signal_buffer[id] is not empty
				if len(to_server_signal_buffers[streamId]) > 0 {
					signals_to_send <- (to_server_signal_buffers[streamId])
					to_server_signal_buffers[streamId] = make([]rtc.Signal, 0)
					return
				}

				time.Sleep(time.Second * 1)
			}
		}()

		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		default:
		}

		signals_to_send_obj := <-signals_to_send

		//offers come before candidates
		sortedSignals := make([]rtc.Signal, 0)
		for _, signal := range signals_to_send_obj {
			if signal.Type == "offer" {
				sortedSignals = append(sortedSignals, signal)
			}
		}
		for _, signal := range signals_to_send_obj {
			if signal.Type == "candidate" {
				sortedSignals = append(sortedSignals, signal)
			}
		}

		json, _ := json.Marshal(sortedSignals)
		return c.String(http.StatusOK, string(json))
	})

	// Server route
	g.POST("/signal/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		signals := utils.ParseBody[[]rtc.Signal](c)

		log.Info().
			Str("method", "POST").
			Str("streamId", streamId).
			Msg("server called /signal/:streamId/internal")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
		if directConnect {
			for _, signal := range signals.Value {
				viewerId := signal.ViewerId
				if to_viewer_signal_buffers[streamId] == nil {
					to_viewer_signal_buffers[streamId] = make(map[string][]rtc.Signal, 0)
				}

				if to_viewer_signal_buffers[streamId][viewerId] == nil {
					to_viewer_signal_buffers[streamId][viewerId] = make([]rtc.Signal, 0)
				}

				to_viewer_signal_buffers[streamId][viewerId] =
					append(to_viewer_signal_buffers[streamId][viewerId], signal)

			}
		} else {

			sc := clientConnectionManager.GetConnection(streamId)
			for _, signal := range signals.Value {
				sc.Signal(signal)
			}
		}

		return c.String(http.StatusOK, "OK")
	})

}
