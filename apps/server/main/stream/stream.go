package stream

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/reactivex/rxgo/v2"
	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func remove(s []chan rxgo.Item, i int) []chan rxgo.Item {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

var channels = make([]chan rxgo.Item, 0)

func CreateVideoCapture() rxgo.Observable {

	bitrate, hasEnv := os.LookupEnv("BITRATE")
	if !hasEnv {
		log.Info().Msg("No bitrate specified, defaulting to 5Mbps")
		bitrate = "5242880"
	}

	resolution, hasEnv := os.LookupEnv("RESOLUTION")
	if !hasEnv {
		log.Info().Msg("No resolution specified, defaulting to 1280x720")
		resolution = "1280x720"
	}
	sizes := strings.Split(resolution, "x")

	framerate, hasEnv := os.LookupEnv("FRAMERATE")
	if !hasEnv {
		log.Info().Msg("No framerate specified, defaulting to 30fps")
		framerate = "30"
	}

	threads, hasEnv := os.LookupEnv("THREADS")
	if !hasEnv {
		log.Info().Msg("No threads specified, defaulting to 2")
		threads = "2"
	}

	frames_ch := make(chan rxgo.Item)
	pipelinearr := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		"show-cursor=0",
		"!",
		"d3d11convert",
		"!",
		"d3d11download",
		"!",
		"queue",
		"!",
		//Optimize for framerate
		"vp8enc",
		"threads=" + threads,
		"deadline=100",
		"max-quantizer=40",
		"min-quantizer=10",
		"max-intra-bitrate=" + bitrate,
		"target-bitrate=" + bitrate,
		//"keyframe-max-dist=10",
		"!",

		fmt.Sprintf("video/x-vp8,framerate=%s/1,width=%s,height=%s", framerate, sizes[0], sizes[1]),
		"!",

		"appsink",
		"name=appsink",
	}

	pipelineString := strings.Join(pipelinearr, " ")

	gst.Init(nil)
	pipeline, err := gst.NewPipelineFromString(pipelineString)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	sink_el, _ := pipeline.GetElementByName("appsink")

	sink := app.SinkFromElement(sink_el)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	var frames = 0
	var buffer_len = int64(0)

	//every second, print stats

	go func() {
		for {

			if pipeline.GetState() != gst.StatePlaying {
				time.Sleep(time.Second)
				frames = 0
				buffer_len = 0
				continue
			}

			time.Sleep(time.Second)
			per_buffer := buffer_len / int64(frames)

			log.Info().
				Int("fps", frames).
				Int("frame_size_kb", int(per_buffer/1024)).
				Int("bitrate_kb", int(buffer_len/1024)).
				Send()

			frames = 0
			buffer_len = 0
		}
	}()

	sink.SetCallbacks(&app.SinkCallbacks{

		NewSampleFunc: func(sink *app.Sink) gst.FlowReturn {

			sample := sink.PullSample()
			if sample == nil {
				return gst.FlowEOS
			}

			buffer := sample.GetBuffer()
			if buffer == nil {
				return gst.FlowError
			}

			len := buffer.GetSize()

			frames++
			buffer_len += len

			select {
			case frames_ch <- rxgo.Of(buffer):
				break
			default:
				break
			}

			return gst.FlowOK
		},
	})

	go func() {

		for {
			val := <-frames_ch

			for i := 0; i < len(channels); i++ {
				select {
				case channels[i] <- val:
				default:
					channels = remove(channels, i)
					log.Info().Int("viewers", len(channels)).Msg("Viewer disconnected")
					if len(channels) == 0 {
						pipeline.SetState(gst.StatePaused)
					}
				}
			}

		}

	}()

	return rxgo.Defer([]rxgo.Producer{func(_ context.Context, ch chan<- rxgo.Item) {
		pipeline.SetState(gst.StatePlaying)
		channel := make(chan rxgo.Item)
		channels = append(channels, channel)
		log.Info().Int("viewers", len(channels)).Msg("Viewer connected")
		for {
			val := <-channel
			ch <- val
		}
	}})
}

func CreateAudioCapture() {

}
