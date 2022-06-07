package rtc

import (
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

func SetupApi() *webrtc.API {
	engine := &webrtc.MediaEngine{}

	// Register Interceptors
	i := &interceptor.Registry{}

	err := webrtc.RegisterDefaultInterceptors(engine, i)
	if err != nil {
		panic(err)
	}
	fb := []webrtc.RTCPFeedback{}
	err = engine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH264,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
			RTCPFeedback: fb,
		},
		PayloadType: 102,
	},
		webrtc.RTPCodecTypeVideo)

	if err != nil {
		panic(err)
	}

	err = engine.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     webrtc.MimeTypeVP8,
				ClockRate:    90000,
				Channels:     0,
				SDPFmtpLine:  "",
				RTCPFeedback: fb,
			},
			PayloadType: 96,
		},
		webrtc.RTPCodecTypeVideo,
	)

	if err != nil {
		panic(err)
	}

	err = engine.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeOpus,
				ClockRate: 48000,
				Channels:  2, SDPFmtpLine: "useinbandfec=1",
				RTCPFeedback: fb,
			},
			PayloadType: 111,
		},
		webrtc.RTPCodecTypeAudio)

	if err != nil {
		panic(err)
	}

	return webrtc.NewAPI(
		webrtc.WithMediaEngine(engine),
		webrtc.WithInterceptorRegistry(i),
	)

}
