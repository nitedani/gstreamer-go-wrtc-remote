package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func WinVP8Pipeline() string {
	config := GetConfig()
	framerate := config.Framerate
	width := config.ResolutionX
	height := config.ResolutionY
	bitrate := config.Bitrate
	threads := config.Threads

	frames_num, _ := strconv.Atoi(framerate)

	pipelinearr_vp8 := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		//"show-cursor=0",
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
	return strings.Join(pipelinearr_vp8, " ")
}

func WinOpenH264Pipeline() string {
	config := GetConfig()
	framerate := config.Framerate
	width := config.ResolutionX
	height := config.ResolutionY
	bitrate := config.Bitrate
	threads := config.Threads
	frames_num, _ := strconv.Atoi(framerate)
	pipelinearr_openh264 := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		//"show-cursor=0",
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
	return strings.Join(pipelinearr_openh264, " ")
}

func WinNvH264Pipeline() string {

	config := GetConfig()
	framerate := config.Framerate
	width := config.ResolutionX
	height := config.ResolutionY
	bitrate := config.Bitrate
	frames_num, _ := strconv.Atoi(framerate)
	bitrate_int, _ := strconv.Atoi(bitrate)
	pipelinearr_nvenc := []string{
		"d3d11screencapturesrc",
		"monitor-index=0",
		//"show-cursor=0",
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

		"h264parse",
		"config-interval=-1",
		//"update-timecode=true",
		"!",

		"appsink",
		"name=appsink",
	}
	return strings.Join(pipelinearr_nvenc, " ")
}

func WinOpusPipeline() string {
	pipelinearr := []string{
		"wasapisrc",
		"low-latency=true",
		"loopback=true",
		"!",
		"audioconvert",
		"!",
		"queue2",
		"max-size-buffers=0",
		"!",
		"opusenc",
		"max-payload-size=1500",
		"bitrate=128000",
		"!",
		"appsink",
		"name=appsink",
	}

	return strings.Join(pipelinearr, " ")
}
