package extract

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func CreateTCPImageSender(target string, tuner string) (error, *TCPImageSender) {
	// Resolve Address

	imgC := make(chan *extractedImage)
	conn, err := net.Dial("tcp", target)
	if err != nil {
		return err, nil
	}

	return nil, &TCPImageSender{&conn, imgC, tuner}
}

func (s *TCPImageSender) SendImage(img *extractedImage) {
	s.imgC <- img
}

func (s *TCPImageSender) Send() <-chan error {

	var err error
	w := bufio.NewWriter(*s.conn)
	for {
		img := <-s.imgC

		if debug {
			log.Printf("Tuner:%s Size:%d ts: %.3f kf:%d\n", s.tuner, len(img.buf), img.ts, img.isKeyFrame)
		}
		_, err = fmt.Fprintf(w, "CHNL %s %d %.3f 0 0\r\n", s.tuner, len(img.buf), img.ts)
		if err != nil {
			log.Fatalf("error writing header %s\n", err)
			continue
		}
		_, err = w.Write(img.buf)
		if err != nil {
			log.Fatalf("error writing image data %s\n", err)
			continue
		}
		w.Flush()
	}

	return nil
}
