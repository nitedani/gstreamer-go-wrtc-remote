package stream

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"signaling/main/utils"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog/log"
)

type Signal struct {
	ViewerId  string `json:"viewerId"`
	Type      string `json:"type"`
	Candidate any    `json:"candidate"`
	SDP       string `json:"sdp"`
}

func randomStr() string {
	n := 5
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}

/////////////////////////////streamId///viewerId//signals
var to_server_signal_buffers = make(map[string][]Signal, 0)

/////////////////////////////streamId///viewerId//signals
var to_viewer_signal_buffers = make(map[string]map[string][]Signal, 0)

func getViewerId(c echo.Context) string {
	id_cookie, err := c.Cookie("connection_id")
	if err != nil {
		panic(err)
	}
	return id_cookie.Value
}

type ICEServer struct {
	URLs           []string    `json:"urls"`
	Username       string      `json:"username,omitempty"`
	Credential     interface{} `json:"credential,omitempty"`
	CredentialType string      `json:"credentialType,omitempty"`
}

func StartSignalingServer(g *echo.Group) {

	iceServers := make([]ICEServer, 0)
	turn_server_url, hasEnv := os.LookupEnv("TURN_SERVER_URL")
	if hasEnv && turn_server_url != "" {
		iceServer := ICEServer{
			URLs: []string{turn_server_url},
		}
		turn_server_username := os.Getenv("TURN_SERVER_USERNAME")
		if turn_server_username != "" {
			iceServer.Username = turn_server_username
		}
		turn_server_password := os.Getenv("TURN_SERVER_PASSWORD")
		if turn_server_password != "" {
			iceServer.Credential = turn_server_password
			iceServer.CredentialType = "password"
		}
		iceServers = append(iceServers, iceServer)
	}

	stun_server_url, hasEnv := os.LookupEnv("STUN_SERVER_URL")
	if hasEnv && stun_server_url != "" {
		iceServer := ICEServer{
			URLs: []string{stun_server_url},
		}
		stun_server_username := os.Getenv("STUN_SERVER_USERNAME")
		if stun_server_username != "" {
			iceServer.Username = stun_server_username
		}
		stun_server_password := os.Getenv("STUN_SERVER_PASSWORD")
		if stun_server_password != "" {
			iceServer.Credential = stun_server_password
			iceServer.CredentialType = "password"
		}
		iceServers = append(iceServers, iceServer)
	}

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
		signals_to_send := make(chan []Signal, 0)

		log.Info().
			Str("method", "GET").
			Str("viewerId", viewerId).
			Str("streamId", streamId).
			Msg("client called /signal/:streamId")

		go func() {
			now := time.Now()
			for {
				//if 20 seconds passed, return empty array
				if time.Since(now) > 20*time.Second {
					signals_to_send <- make([]Signal, 0)
					return
				}

				if to_viewer_signal_buffers[streamId] == nil {
					to_viewer_signal_buffers[streamId] = make(map[string][]Signal, 0)
				}

				if to_viewer_signal_buffers[streamId][viewerId] == nil {
					to_viewer_signal_buffers[streamId][viewerId] = make([]Signal, 0)
				}

				// wait until signal_buffer[id] is not empty
				if len(to_viewer_signal_buffers[streamId][viewerId]) > 0 {
					signals_to_send <- (to_viewer_signal_buffers[streamId][viewerId])
					to_viewer_signal_buffers[streamId][viewerId] = make([]Signal, 0)
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

		if to_server_signal_buffers[streamId] == nil {
			return c.String(http.StatusNotFound, "stream not found")
		}
		signal := utils.ParseBody[Signal](c)
		signal.Value.ViewerId = viewerId
		to_server_signal_buffers[streamId] = append(
			to_server_signal_buffers[streamId],
			signal.Value)
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
		signals_to_send := make(chan []Signal, 0)

		log.Info().
			Str("method", "GET").
			Str("streamId", streamId).
			Msg("server called /signal/:streamId/internal")

		if to_server_signal_buffers[streamId] == nil {
			to_server_signal_buffers[streamId] = make([]Signal, 0)
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
					signals_to_send <- make([]Signal, 0)
					return
				}

				// wait until signal_buffer[id] is not empty
				if len(to_server_signal_buffers[streamId]) > 0 {
					signals_to_send <- (to_server_signal_buffers[streamId])
					to_server_signal_buffers[streamId] = make([]Signal, 0)
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
		sortedSignals := make([]Signal, 0)
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
		signals := utils.ParseBody[[]Signal](c)

		log.Info().
			Str("method", "POST").
			Str("streamId", streamId).
			Msg("server called /signal/:streamId/internal")

		for _, signal := range signals.Value {

			viewerId := signal.ViewerId
			if to_viewer_signal_buffers[streamId] == nil {
				to_viewer_signal_buffers[streamId] = make(map[string][]Signal, 0)
			}

			if to_viewer_signal_buffers[streamId][viewerId] == nil {
				to_viewer_signal_buffers[streamId][viewerId] = make([]Signal, 0)
			}

			to_viewer_signal_buffers[streamId][viewerId] =
				append(to_viewer_signal_buffers[streamId][viewerId], signal)

		}

		return c.String(http.StatusOK, "OK")
	})

}
