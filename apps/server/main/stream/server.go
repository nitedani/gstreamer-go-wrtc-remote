package stream

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"signaling/main/rtc"
	"signaling/main/utils"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog/log"
)

type NewStreamBody struct {
	IsDirectConnect bool `json:"isDirectConnect"`
	IsPrivate       bool `json:"isPrivate"`
}

var runId = utils.RandomStr()

func StartSignalingServer(g *echo.Group) {

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

	// Viewer route
	g.GET("/signal/:streamId", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		viewerId := utils.GetViewerId(c)

		log.Info().
			Str("method", "GET").
			Str("viewerId", viewerId).
			Str("streamId", streamId).
			Msg("viewer called /signal/:streamId")

		streamId = streamId + runId
		stream := streamManager.GetStream(streamId)
		json, _ := json.Marshal(<-stream.GetSignalsForViewer(viewerId))
		return c.String(http.StatusOK, string(json))
	})

	// Viewer route
	g.POST("/signal/:streamId", func(c echo.Context) error {

		streamId := c.PathParam("streamId")
		viewerId := utils.GetViewerId(c)

		log.Info().
			Str("method", "POST").
			Str("viewerId", viewerId).
			Str("streamId", streamId).
			Msg("viewer called /signal/:streamId")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId

		stream := streamManager.GetStream(streamId)

		if stream == nil || !stream.IsAvailable() {
			return c.String(http.StatusNotFound, "stream not found")
		}

		signals := utils.ParseBody[[]rtc.Signal](c)

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
				// build the pipeline: capture client -> server -> viewer
				stream.Connection.ConnectTo(viewerConnection)

			}

			for _, signal := range signals.Value {
				signal.ViewerId = viewerId
				// build the connection between the server and viewer
				viewerConnection.Signal(signal)
			}

		}

		return c.String(http.StatusOK, "OK")
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
			return c.String(http.StatusBadRequest, err.Error())
		}
		buffer := bytes.NewBuffer(buf)
		streamManager.SetSnapshot(streamId, buffer)
		return c.String(http.StatusOK, "OK")
	})

	// Client route
	g.POST("/connect/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
		stream := streamManager.NewStream(streamId)
		body := utils.ParseBody[NewStreamBody](c)
		stream.IsDirectConnect = body.Value.IsDirectConnect
		stream.IsPrivate = body.Value.IsPrivate
		return c.String(http.StatusOK, "OK")
	})

	// Client route
	g.GET("/signal/:streamId/internal", func(c echo.Context) error {
		streamId := c.PathParam("streamId")
		isDirectConnect := c.QueryParam("isDirectConnect")
		isPrivate := c.QueryParam("isPrivate")
		// parse bool
		isDirectConnectBool, _ := strconv.ParseBool(isDirectConnect)
		isPrivateBool, _ := strconv.ParseBool(isPrivate)
		signals_to_send := make(chan []rtc.Signal)

		log.Info().
			Str("method", "GET").
			Str("streamId", streamId).
			Msg("client called /signal/:streamId/internal")

		// if the server is restarted, need to force a new connection
		streamId = streamId + runId
		stream := streamManager.GetStream(streamId)
		if stream == nil {
			stream = streamManager.NewStream(streamId)
			stream.IsDirectConnect = isDirectConnectBool
			stream.IsPrivate = isPrivateBool
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

				stream.SignalFromCaptureClient(signal)
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
