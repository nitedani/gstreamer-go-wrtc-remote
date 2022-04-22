package rtc

import (
	"os"
	"strconv"
)

type Configuration struct {
	ICEServers    []ICEServer `json:"iceServers"`
	DirectConnect bool        `json:"directConnect"`
}

func GetRtcConfig() Configuration {
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

	direct_connect, hasEnv := os.LookupEnv("DIRECT_CONNECT")
	if !hasEnv {
		panic("DIRECT_CONNECT not set")
	}
	direct_connect_bool, err := strconv.ParseBool(direct_connect)
	if err != nil {
		panic(err)
	}

	return Configuration{
		ICEServers:    iceServers,
		DirectConnect: direct_connect_bool,
	}
}
