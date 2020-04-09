package extract

import (
	"github.com/3d0c/gmf"
	"image"
	"net"
)

type Extractor struct {
	source                    string
	tuner                     string
	targetVideo               string
	targetAudio               string
	inputCtx                  *gmf.FmtCtx
	videoStream               *gmf.Stream
	audioStream               *gmf.Stream
	targetWidth, targetHeight int
	swsCtx                    *gmf.SwsCtx
	cc                        *gmf.CodecCtx
	acc                       *gmf.CodecCtx
	swrCtx                    *gmf.SwrCtx
	drain                     int
	videoDecoderC             chan *gmf.Packet
	audioDecoderC             chan *gmf.Packet
}

type extractedImage struct {
	buf        []byte
	ts         float64
	isKeyFrame uint8
}

type extractedAudio struct {
	buf []byte
	ts  float64
}

type KeyFrame struct {
	currentKeyFrame *image.Image
	counter         int
}

type ImageProcessor struct {
	lastImageTS  float64
	rate         uint
	keyFrameProc *KeyFrame
	counter      int
}

type AudioProcessor struct {
}

type ImageSender interface {
	Send() <-chan error
	SendImage(img *extractedImage)
}

type EmptyImageSender struct{}

type TCPImageSender struct {
	conn  *net.Conn
	imgC  chan *extractedImage
	tuner string
}

type AudioSender interface {
	Send() <-chan error
	SendAudio(aud *extractedAudio)
}

type FileAudioSender struct {
	filename string
	audC chan *extractedAudio
}

type UDPAudioSender struct {
}
