package extract

import (
	"fmt"
	"github.com/3d0c/gmf"
	"image"
	"io"
	"log"
	"time"
)

func CreateExtractor(source *string, tuner *string, targetVideo *string, targetAudio *string) (error, *Extractor) {
	return nil, &Extractor{
		source:      *source,
		tuner:       *tuner,
		targetVideo: *targetVideo,
		targetAudio: *targetAudio,
	}
}

func (ex *Extractor) initAudio() error {
	var err error

	ex.audioStream, err = ex.inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		return err
	}

	ctx := ex.audioStream.CodecCtx()

	channels := 1
	sampleFmt := ctx.SampleFmt()
	//sampleRate := 16000 // This does not work.
	sampleRate := ctx.SampleRate()

	options := []*gmf.Option{
		{"in_channel_count", ctx.Channels()},
		{"in_sample_rate", ctx.SampleRate()},
		{"in_sample_fmt", ctx.SampleFmt()},
		{"out_channel_count" , channels},
		{"out_sample_rate", sampleRate  },
		{"out_sample_fmt", sampleFmt},
	}
	ex.swrCtx,err = gmf.NewSwrCtx(options,channels,sampleFmt)

	if err != nil {
		return err
	}

	return nil

}

func (ex *Extractor) initVideo() error {
	var err error

	ex.videoStream, err = ex.inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		return err
	}

	codec, err := gmf.FindEncoder(gmf.AV_CODEC_ID_RAWVIDEO)
	if err != nil {
		return err
	}

	ex.cc = gmf.NewCodecCtx(codec)

	ex.cc.SetTimeBase(gmf.AVR{Num: 1, Den: 1})

	ex.targetWidth = 200
	ex.targetHeight = 150

	ex.cc.SetPixFmt(gmf.AV_PIX_FMT_RGBA).SetWidth(ex.targetWidth).SetHeight(ex.targetHeight)
	if codec.IsExperimental() {
		ex.cc.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err := ex.cc.Open(nil); err != nil {
		return err
	}

	// convert source pix_fmt into AV_PIX_FMT_RGBA
	// which is set up by codec context above
	icc := ex.videoStream.CodecCtx()
	if ex.swsCtx, err = gmf.NewSwsCtx(icc.Width(), icc.Height(), icc.PixFmt(), ex.cc.Width(), ex.cc.Height(), ex.cc.PixFmt(), gmf.SWS_BICUBIC); err != nil {
		return err
	}

	return nil
}
func (ex *Extractor) Init() error {
	var err error
	ex.inputCtx, err = gmf.NewInputCtxWithFormatName(ex.source, "mpegts")
	//defer ex.inputCtx.Free()

	if err != nil {
		return err
	}

	fmt.Println(ex.inputCtx.StartTime())

	err = ex.initVideo()
	if err != nil {
		log.Fatal("error initializing video")
		return err
	}
	err = ex.initAudio()
	if err != nil {
		log.Fatal("error initializing audio")
		return err
	}

	return nil
}

func (ex *Extractor) Close() {
	log.Println("Closing")
	if ex.swsCtx != nil {
		ex.swsCtx.Free()
	}

	if ex.cc != nil {
		ex.cc.Free()
		gmf.Release(ex.cc)
	}

	if ex.inputCtx != nil {
		for i := 0; i < ex.inputCtx.StreamsCnt(); i++ {
			st, _ := ex.inputCtx.GetStream(i)
			st.CodecCtx().Free()
			st.Free()
		}

		ex.inputCtx.Free()
	}
}

func (ex *Extractor) extractAudio(sender AudioSender, proc *AudioProcessor) {
	var (
		frames []*gmf.Frame
		frame *gmf.Frame
		err    error
	)

	for {
		pkt := <-ex.audioDecoderC

		if debug || debugAudio {
			fmt.Println("got packet from audio decoder channel")
		}

		frames, err = ex.audioStream.CodecCtx().Decode(pkt)
		if err != nil {
			fmt.Printf("error during decoding - %s %d\n", err, len(frames))
			continue
		}

		if len(frames) == 0 && ex.drain < 0 {
			continue
		}

		for _, f := range frames {

			fmt.Printf("before channels: %d  linesize: %d channel_layout: %d nbsamples:%d\n" ,
				f.Channels() , f.LineSize(0) , f.GetChannelLayout() , f.NbSamples() )

			// covert to mono 16Khz
			frame,err = ex.swrCtx.Convert(f)
			fmt.Println("converted")
			if err!=nil {
				fmt.Println("there's an error")
				fmt.Errorf("error while converting audio %v\n" , err)
				continue
			}


			fmt.Printf("after channels: %d  linesize: %d channel_layout: %d nbsamples:%d\n" ,
				frame.Channels() , frame.LineSize(0) , frame.GetChannelLayout() , frame.NbSamples())

			b := frame.GetRawAudioData(0)

			fmt.Printf("size of audio buf %d \n", len(b) )
			ts := float64(time.Now().UnixNano()) / float64(1000000000)
			aud := &extractedAudio{ b,ts}
			sender.SendAudio(aud)

		}

		if pkt != nil {
			pkt.Free()
			pkt = nil
		}

	}
}

func (ex *Extractor) extractVideo(sender ImageSender, proc *ImageProcessor) {
	var (
		frames []*gmf.Frame
		err    error
	)

	for {
		pkt := <-ex.videoDecoderC

		if debug || debugVideo {
			fmt.Println("got packet from video decoder channel")
		}

		frames, err = ex.videoStream.CodecCtx().Decode(pkt)
		if err != nil {
			fmt.Printf("Fatal error during decoding - %s\n", err)
			break
		}

		// Decode() method doesn't treat EAGAIN and EOF as errors
		// it returns empty frames slice instead. Countinue until
		// input EOF or frames received.
		if len(frames) == 0 && ex.drain < 0 {
			continue
		}

		if frames, err = gmf.DefaultRescaler(ex.swsCtx, frames); err != nil {
			panic(err)
		}

		packets, err := ex.cc.Encode(frames, ex.drain)
		if err != nil {
			log.Fatalf("Error encoding - %s\n", err)
		}
		if len(packets) == 0 {
			return
		}

		for _, p := range packets {
			//fmt.Printf("PTS is %d\n",p.Pts())
			width, height := ex.cc.Width(), ex.cc.Height()

			img := new(image.RGBA)
			img.Pix = p.Data()
			img.Stride = 4 * width
			img.Rect = image.Rect(0, 0, width, height)

			ts := float64(time.Now().UnixNano()) / float64(1000000000)

			err, buf, isKeyFrame := proc.Process(img, ts)

			if err != nil {
				panic(err)
			}

			if buf != nil {
				sender.SendImage(&extractedImage{buf, ts, isKeyFrame})
			}

			p.Free()
		}

		for i, _ := range frames {
			frames[i].Free()
		}

		if pkt != nil {
			pkt.Free()
			pkt = nil
		}

	}
}

func (ex *Extractor) Extract() error {

	var (
		pktCount  int = 0
		err       error
		imgSender ImageSender
		imgProc   *ImageProcessor

		audioProc   *AudioProcessor
		audioSender AudioSender
	)

	ex.drain = -1
	ex.videoDecoderC = make(chan *gmf.Packet, 100)
	ex.audioDecoderC = make(chan *gmf.Packet, 100)

	// start video stuff
	if len(ex.targetVideo) > 0 {
		err, imgSender = CreateTCPImageSender(ex.targetVideo, ex.tuner)
		if err != nil {
			fmt.Printf("Fatal error while creating sender - %s\n", err)
			return err
		}
	} else {
		imgSender = &EmptyImageSender{}
	}

	go imgSender.Send()

	err, imgProc = CreateImageProcessor(5)

	go ex.extractVideo(imgSender, imgProc)

	// start audio stuff
	err , audioSender = CreateFileAudioSender(ex.targetAudio)

	go audioSender.Send()


	go ex.extractAudio(audioSender, audioProc)

	for {
		if ex.drain >= 0 {
			break
		}

		pkt, err := ex.inputCtx.GetNextPacket()
		if err != nil && err != io.EOF {
			if pkt != nil {
				pkt.Free()
			}
			return err
		} else if err != nil && pkt == nil {
			ex.drain = 0
		}

		if pkt != nil {
			pktCount++

			if pkt.StreamIndex() == ex.videoStream.Index() {
				if debug || debugVideo {
					fmt.Println("sending packet to video decoder channel", pktCount)
				}

				ex.videoDecoderC <- pkt
			}

			if pkt.StreamIndex() == ex.audioStream.Index() {
				if debug || debugAudio {
					fmt.Println("sending packet to audio decoder channel", pktCount)
				}

				ex.audioDecoderC <- pkt
			}
		}

	}

	return nil
}
