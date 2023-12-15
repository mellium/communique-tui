package gst

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0

#include "gst.h"

*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

func GstreamerInit() {
	C.gstreamer_init()
	go C.gstreamer_start_mainloop()
}

type ReceivePipeline struct {
	Pipeline *C.GstElement
}

func CreateReceivePipeline(payloadType webrtc.PayloadType, codecName string) (*ReceivePipeline, error) {
	pipelineStr := "appsrc format=time is-live=true do-timestamp=true name=src ! application/x-rtp"
	switch strings.ToLower(codecName) {
	case "vp8":
		pipelineStr += fmt.Sprintf(", payload=%d, encoding-name=VP8-DRAFT-IETF-01 ! rtpvp8depay ! decodebin ! videoconvert ! autovideosink", payloadType)
	case "opus":
		pipelineStr += fmt.Sprintf(", payload=%d, encoding-name=OPUS ! rtpopusdepay ! decodebin ! audioconvert ! audioresample ! autoaudiosink", payloadType)
	case "vp9":
		pipelineStr += " ! rtpvp9depay ! decodebin ! videoconvert ! autovideosink"
	case "h264":
		pipelineStr += " ! rtph264depay ! decodebin ! videoconvert ! autovideosink"
	case "g722":
		pipelineStr += " clock-rate=8000 ! rtpg722depay ! decodebin ! audioconvert ! audioresample ! autoaudiosink"
	default:
		return nil, errors.New("Unhandled codec " + codecName)
	}

	pipelineStrUnsafe := C.CString(pipelineStr)
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))
	return &ReceivePipeline{Pipeline: C.gstreamer_receive_create_pipeline(pipelineStrUnsafe)}, nil
}

func (p *ReceivePipeline) Start() {
	C.gstreamer_receive_start_pipeline(p.Pipeline)
}

func (p *ReceivePipeline) Stop() {
	C.gstreamer_receive_stop_pipeline(p.Pipeline)
}

func (p *ReceivePipeline) Push(buffer []byte) {
	b := C.CBytes(buffer)
	defer C.free(b)
	C.gstreamer_receive_push_buffer(p.Pipeline, b, C.int(len(buffer)))
}

func (p *ReceivePipeline) Free() {
	C.gstreamer_free_pipeline(p.Pipeline)
}

type SendPipeline struct {
	Pipeline  *C.GstElement
	tracks    []*webrtc.TrackLocalStaticSample
	id        int
	codecName string
	clockRate float32
}

var (
	pipelines     = make(map[int]*SendPipeline)
	pipelinesLock sync.Mutex
)

const (
	videoClockRate = 90000
	audioClockRate = 48000
	pcmClockRate   = 8000
)

func CreateSendPipeline(codecName string, tracks []*webrtc.TrackLocalStaticSample) (*SendPipeline, error) {
	pipelineStr := "appsink name=appsink"
	pipelineVideoSrc := "autovideosrc ! video/x-raw, width=320, height=240 ! videoconvert"
	pipelineAudioSrc := "autoaudiosrc ! queue ! audioconvert"
	var clockRate float32

	switch codecName {
	case "vp8":
		pipelineStr = pipelineVideoSrc + " ! vp8enc error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 ! tee name=t t. ! queue ! vp8dec ! decodebin ! videoconvert ! autovideosink t. ! queue ! appsink name=appsink"
		clockRate = videoClockRate
	case "vp9":
		pipelineStr = pipelineVideoSrc + " ! vp9enc ! " + pipelineStr
		clockRate = videoClockRate
	case "h264":
		pipelineStr = pipelineVideoSrc + " ! video/x-raw,format=I420 ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! video/x-h264,stream-format=byte-stream ! " + pipelineStr
		clockRate = videoClockRate
	case "opus":
		pipelineStr = pipelineAudioSrc + " ! opusenc ! " + pipelineStr
		clockRate = audioClockRate
	case "g722":
		pipelineStr = pipelineAudioSrc + " ! avenc_g722 ! " + pipelineStr
		clockRate = audioClockRate
	case "pcmu":
		pipelineStr = pipelineAudioSrc + " ! audio/x-raw, rate=8000 ! mulawenc ! " + pipelineStr
		clockRate = pcmClockRate
	case "pcma":
		pipelineStr = pipelineAudioSrc + " ! audio/x-raw, rate=8000 ! alawenc ! " + pipelineStr
		clockRate = pcmClockRate
	case "videotest":
		pipelineStr = "autovideosrc ! video/x-raw, width=320, height=240 ! videoconvert ! vp8enc error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 ! queue ! appsink name=appsink"
		clockRate = videoClockRate
	case "audiotest":
		pipelineStr = "autoaudiosrc ! queue ! audioconvert ! opusenc ! " + pipelineStr
		clockRate = audioClockRate
	default:
		return nil, errors.New("Unhandled codec " + codecName)
	}

	pipelineStrUnsafe := C.CString(pipelineStr)
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))

	pipelinesLock.Lock()
	defer pipelinesLock.Unlock()

	pipeline := &SendPipeline{
		Pipeline:  C.gstreamer_send_create_pipeline(pipelineStrUnsafe),
		tracks:    tracks,
		id:        len(pipelines),
		codecName: codecName,
		clockRate: clockRate,
	}

	pipelines[pipeline.id] = pipeline
	return pipeline, nil
}

func (p *SendPipeline) Start() {
	C.gstreamer_send_start_pipeline(p.Pipeline, C.int(p.id))
}

func (p *SendPipeline) Stop() {
	C.gstreamer_send_stop_pipeline(p.Pipeline)
}

func (p *SendPipeline) Free() {
	C.gstreamer_free_pipeline(p.Pipeline)
}

//export goHandlePipelineBuffer
func goHandlePipelineBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, pipelineID C.int) {
	pipelinesLock.Lock()
	pipeline, ok := pipelines[int(pipelineID)]
	pipelinesLock.Unlock()

	if ok {
		for _, t := range pipeline.tracks {
			if err := t.WriteSample(media.Sample{Data: C.GoBytes(buffer, bufferLen), Duration: time.Duration(duration)}); err != nil {
				panic(err)
			}
		}
	} else {
		fmt.Printf("discarding buffer, no pipeline with id %d", int(pipelineID))
	}
	C.free(buffer)
}
