package extract

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"
)

// Check if image is a new keyframe and converts it to jpeg
// Check if image difference is enough
// TODO make it a real frame.
//

func CreateImageProcessor(rate uint) (error, *ImageProcessor) {

	proc := &ImageProcessor{rate: rate}
	proc.keyFrameProc = &KeyFrame{}

	return nil, proc
}

func (ip *ImageProcessor) CheckRateThreshold(ts float64) bool {
	if ip.lastImageTS == 0 {
		ip.lastImageTS = ts
		return true
	}
	// TODO fix this crappy calc
	currentRate := uint(1 / (ts - ip.lastImageTS))
	if currentRate < ip.rate {
		return false
	}

	return true
}

func (ip *ImageProcessor) Process(img image.Image, ts float64) (error, []byte, uint8) {

	if debug {
		log.Println("processing image")
	}

	ip.counter++

	err, isKeyFrame := ip.keyFrameProc.IsKeyFrame(img)

	if isKeyFrame == 0 {
		// not keyframe
		if ip.CheckRateThreshold(ts) == false {
			log.Printf("skipping image %d\n", ip.counter)

			return nil, nil, 0
		}
	} else {
		// set last ts
		ip.lastImageTS = ts
	}

	buf := new(bytes.Buffer)

	err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		return err, nil, 0
	}

	return nil, buf.Bytes(), 0
}

func (kf *KeyFrame) IsKeyFrame(img image.Image) (error, uint8) {
	return nil, 0
}
