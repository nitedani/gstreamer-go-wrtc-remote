package stream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"signaling/main/rtc"
	"signaling/main/utils"

	socketio "github.com/googollee/go-socket.io"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog/log"
)

type NewStreamBody struct {
	IsDirectConnect bool `json:"isDirectConnect"`
	IsPrivate       bool `json:"isPrivate"`
}

var runId = utils.RandomStr()

type ViewerSocketContext struct {
	StreamId string `json:"streamId"`
	ViewerId string `json:"viewerId"`
}

type StreamerSocketContext struct {
	StreamId string `json:"streamId"`
}

type ConnectionEvent struct {
	Type        string `json:"type"`
	ViewerId    string `json:"viewerId"`
	ViewerCount int    `json:"viewerCount"`
}

func StartSignalingServer(g *echo.Group, ss *socketio.Server) {

	config := rtc.GetRtcConfig()
	iceServers := config.ICEServers
	directConnect := config.DirectConnect

	streamManager := NewStreamManager(g)

	g.GET("/streams", func(c echo.Context) error {
		return c.JSON(http.StatusOK, streamManager.ListStreams())
	})

	g.GET("/snapshot/:streamId", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		log.Info().
			Str("method", "GET").
			Str("streamId", streamId).
			Msg("viewer called /snapshot/:streamId")

		streamId = streamId + runId
		snapshot := streamManager.GetSnapshot(streamId)
		return c.Blob(http.StatusOK, "image/jpg", snapshot.Bytes())
	})

	g.GET("/ice-config", func(c echo.Context) error {
		log.Info().
			Msg("client called /ice-config")

		return c.JSON(http.StatusOK, iceServers)
	})

	// Viewer route
	g.POST("/connect", func(c echo.Context) error {
		viewerId := utils.RandomStr()
		log.Info().
			Str("viewerId", viewerId).
			Msg("client called /connect")

		c.SetCookie(&http.Cookie{
			Name:  "connection_id",
			Value: viewerId,
		})
		return c.String(http.StatusOK, viewerId)
	})

	ss.OnConnect("/", func(s socketio.Conn) error {

		url := s.URL()
		key := url.Query().Get("streamKey")
		streamId := url.Query().Get("streamId")
		streamId = streamId + runId

		log.Error().
			Str("url", url.String()).
			Str("streamId", streamId).
			Str("streamKey", key).
			Msg("viewer connected")

		stream := streamManager.GetStream(streamId)

		if stream == nil {
			s.Close()
			return nil
		}

		stream.OnViewerConnected(func(connectionId string) {
			event := ConnectionEvent{
				Type:        "viewer_connected",
				ViewerId:    connectionId,
				ViewerCount: stream.GetViewerCount(),
			}
			go s.Emit("conn_ev", event)
		})
		stream.OnViewerDisconnected(func(connectionId string) {
			event := ConnectionEvent{
				Type:        "viewer_disconnected",
				ViewerId:    connectionId,
				ViewerCount: stream.GetViewerCount(),
			}
			go s.Emit("conn_ev", event)
		})

		if key == "" {
			viewerId := utils.RandomStr()
			s.SetContext(&ViewerSocketContext{StreamId: streamId, ViewerId: viewerId})
			s.Join(viewerId)
			s.Join(streamId + "_viewers")
			fmt.Println("viewer connected:", viewerId)

		} else {
			// the streamer can listen for events for their own stream
			// for now, only viewer connect/disconenct events are supported
			//TODO: validate the key, for now there is no data returned
			s.SetContext(&StreamerSocketContext{StreamId: streamId})
			s.Join(streamId)
			fmt.Println("streamer connected:", streamId)
		}
		return nil
	})

	// Viewer route
	ss.OnEvent("/", "signal", func(s socketio.Conn, msg string) error {

		signals := utils.ParseJson[[]rtc.Signal](msg)
		ctx := s.Context().(*ViewerSocketContext)
		streamId := ctx.StreamId
		viewerId := ctx.ViewerId

		stream := streamManager.GetStream(streamId)

		log.Info().
			Str("method", "signal").
			Str("viewerId", viewerId).
			Str("streamId", streamId).
			Msg("viewer called /signal/:streamId")

		if stream == nil || !stream.IsAvailable() {
			return nil
		}

		if directConnect || stream.IsDirectConnect {
			log.Info().Msg("direct connect")
			for _, signal := range signals.Value {
				signal.ViewerId = viewerId
				// forward the signal to the capture client
				// build the connection between the viewer and capture client

				stream.SignalToCaptureClient(signal)
			}
		} else {
			// connect the server to the capture client
			if stream.Connection == nil {
				stream.ConnectClient()
			}

			viewerConnection := stream.GetViewer(viewerId)
			if viewerConnection == nil {

				viewerConnection = stream.NewViewer(viewerId)

				viewerConnection.OnSignal(func(signal rtc.Signal) {
					go s.Emit("signal", signal)
				})
				// build the pipeline: capture client -> server -> viewer
				stream.Connection.ConnectTo(viewerConnection)

			}

			for _, signal := range signals.Value {
				signal.ViewerId = viewerId
				// build the connection between the server and viewer
				viewerConnection.Signal(signal)
			}
		}

		return nil
	})

	// Client route
	g.POST("/snapshot/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		log.Info().
			Str("method", "POST").
			Str("streamId", streamId).
			Msg("client called /snapshot/:streamId/internal")

		streamId = streamId + runId

		buf, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return errors.New("failed to read body")
		}
		buffer := bytes.NewBuffer(buf)
		streamManager.SetSnapshot(streamId, buffer)

		return c.String(http.StatusOK, "OK")
	})

	// Client route
	g.POST("/conn-evt/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		log.Info().
			Str("method", "POST").
			Str("streamId", streamId).
			Msg("client called /conn-evt/:streamId/internal")

		streamId = streamId + runId

		stream := streamManager.GetStream(streamId)

		body := utils.ParseBody[ConnectionEvent](c)

		event := body.Value

		stream.OnClientConnectionEvent(event)

		return c.String(http.StatusOK, "OK")
	})

	// Client route
	g.POST("/connect/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
		body := utils.ParseBody[NewStreamBody](c)
		isDirectConnect := body.Value.IsDirectConnect
		isPrivate := body.Value.IsPrivate
		streamManager.NewStream(streamId, isDirectConnect, isPrivate)

		return c.String(http.StatusOK, "OK")
	})

	// Client route
	g.GET("/signal/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")

		signals_to_send := make(chan []rtc.Signal)

		log.Info().
			Str("method", "GET").
			Str("streamId", streamId).
			Msg("client called /signal/:streamId/internal")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
		stream := streamManager.GetStream(streamId)
		if stream == nil {
			return c.String(http.StatusNotFound, "{\"message\":\"stream not found\"}")
		}
		go func() {
			for {
				select {
				case <-c.Request().Context().Done():
					return
				case signals := <-stream.GetSignalsForCaptureClient():

					signals_to_send <- signals
				}
			}
		}()
		json, _ := json.Marshal(utils.SortSignals(<-signals_to_send))
		return c.String(http.StatusOK, string(json))
	})

	// Client route
	g.POST("/signal/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		signals := utils.ParseBody[[]rtc.Signal](c)

		log.Info().
			Str("method", "POST").
			Str("streamId", streamId).
			Msg("client called /signal/:streamId/internal")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
		stream := streamManager.GetStream(streamId)

		if directConnect || stream.IsDirectConnect {
			for _, signal := range signals.Value {
				viewerId := signal.ViewerId
				ss.BroadcastToRoom("/", viewerId, "signal", signal)
			}
		} else {
			cc := stream.Connection
			for _, signal := range signals.Value {
				cc.Signal(signal)
			}
		}

		return c.String(http.StatusOK, "OK")
	})

}
