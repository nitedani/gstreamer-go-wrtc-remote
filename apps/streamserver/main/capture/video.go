package capture

import (
	"context"
	"fmt"
	"os"
	"server/main/utils"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tinyzimmer/go-gst/gst"
	"github.com/tinyzimmer/go-gst/gst/app"
)

var video_channels = make(map[string]*MediaChannel, 0)

func CreateVideoCapture() func() chan *gst.Buffer {

	config := utils.GetConfig()
	framerate := config.Framerate
	width := config.ResolutionX
	height := config.ResolutionY
	bitrate := config.Bitrate
	encoder := config.Encoder
	threads := config.Threads

	frames_ch := make(chan *gst.Buffer)

	frames_num, _ := strconv.Atoi(framerate)

	pipelinearr_vp8 := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		"show-cursor=0",
		"!",

		"d3d11convert",
		"!",

		"d3d11download",
		"!",

		//fmt.Sprintf("video/x-raw,framerate=%s/1", framerate),
		//"!",

		"queue2",
		"max-size-buffers=0",
		"max-size-bytes=0",
		"max-size-time=" + strconv.Itoa((1000000000/frames_num)*2),
		"!",

		//Optimize for framerate
		"vp8enc",
		"threads=" + threads,
		"deadline=1",
		"max-quantizer=40",
		"min-quantizer=4",
		"max-intra-bitrate=" + bitrate,
		"target-bitrate=" + bitrate,
		"!",

		fmt.Sprintf("video/x-vp8,framerate=%s/1,width=%s,height=%s", framerate, width, height),
		"!",

		"appsink",
		"name=appsink",
	}

	pipelinearr_h264 := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		"show-cursor=0",
		"!",

		"d3d11convert",
		"!",

		"d3d11download",
		"!",

		fmt.Sprintf("video/x-raw,framerate=%s/1,width=%s,height=%s", framerate, width, height),
		"!",

		"queue2",
		"max-size-buffers=0",
		"max-size-bytes=0",
		"max-size-time=" + strconv.Itoa((1000000000/frames_num)*2),
		"!",

		//Optimize for framerate
		"openh264enc",
		"enable-frame-skip=true",
		"deblocking=1",
		"bitrate=" + bitrate,
		"complexity=0",
		"multi-thread=" + threads,
		"qp-max=40",
		"slice-mode=5",
		"!",

		//fmt.Sprintf("video/x-h264,framerate=%s/1,width=%s,height=%s", framerate, sizes[0], sizes[1]),
		//"!",

		"appsink",
		"name=appsink",
	}

	bitrate_int, _ := strconv.Atoi(bitrate)

	pipelinearr_nvenc := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		"show-cursor=0",
		"!",

		"d3d11convert",
		"!",

		"d3d11download",
		"!",

		fmt.Sprintf("video/x-raw,format=NV12,framerate=%s/1,width=%s,height=%s", framerate, width, height),
		"!",

		"queue2",
		"max-size-buffers=0",
		"max-size-bytes=0",
		"max-size-time=" + strconv.Itoa((1000000000/frames_num)*2),
		"!",

		//Optimize for framerate
		"nvh264enc",
		"preset=5",
		"rc-mode=5",
		"zerolatency=true",
		//Convert bitrate from bits to kbits
		"bitrate=" + strconv.Itoa(bitrate_int/1024),
		"!",

		"appsink",
		"name=appsink",
	}

	var pipelineArr []string

	switch encoder {
	case "vp8":
		pipelineArr = pipelinearr_vp8
	case "h264":
		pipelineArr = pipelinearr_h264
	case "nvenc":
		pipelineArr = pipelinearr_nvenc
	default:
		log.Fatal().Msg("Invalid encoder specified")
	}

	pipelineString := strings.Join(pipelineArr, " ")

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

	go func() {
		for {

			if pipeline.GetState() != gst.StatePlaying || frames == 0 {
				time.Sleep(time.Second)
				continue
			}

			time.Sleep(time.Second)
			per_buffer := buffer_len / int64(frames)

			log.Info().
				Int("video_framerate", frames).
				Int("video_frame_size_kb", int(per_buffer/1024)).
				Int("video_bitrate_kb", int(buffer_len/1024)).
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
			case frames_ch <- buffer:
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
			for _, channel := range video_channels {
				if channel.IsDone || channel.Writing {
					continue
				}
				channel.Writing = true
				go func(channel *MediaChannel) {
					select {
					case <-channel.Context.Done():
						return
					case channel.Channel <- val:
						channel.Writing = false
						return
					case <-time.After(time.Millisecond * 100):
						channel.IsDone = true
						channel.Cancel()

					}
				}(channel)
			}
		}
	}()

	return func() chan *gst.Buffer {
		ctx, cancel := context.WithCancel(context.Background())

		id := utils.RandomStr()

		mediaChannel := MediaChannel{
			ID:      id,
			Channel: make(chan *gst.Buffer),
			Context: ctx,
			Cancel:  cancel,
			IsDone:  false,
			Writing: false,
		}

		video_channels[id] = &mediaChannel

		if condition := pipeline.GetState() != gst.StatePlaying; condition {
			log.Info().Int("viewers", len(video_channels)).Msg("Starting video capture")
			pipeline.SetState(gst.StatePlaying)
		}

		go func() {
			<-mediaChannel.Context.Done()
			delete(video_channels, id)
			log.Info().Int("viewers", len(video_channels)).Msg("Closing video channel")
			if len(video_channels) == 0 {
				log.Info().Msg("Pausing screen capture")
				pipeline.SetState(gst.StatePaused)
			}
		}()

		return mediaChannel.Channel
	}
}
