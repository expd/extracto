package extract

import (
	"bufio"
	"fmt"
	"os"
)

func (s *EmptyImageSender) SendImage(img *extractedImage) {
}

func (s *EmptyImageSender) Send() <-chan error {

	return nil
}

func CreateFileAudioSender(filename string) (error, *FileAudioSender) {
	audC := make(chan *extractedAudio)
	return nil, &FileAudioSender{filename, audC}
}

func (s *FileAudioSender) SendAudio(aud *extractedAudio) {
	s.audC <- aud
}

func (s *FileAudioSender) Send() <-chan error {
	// open filename and start writing

	f, err := os.Create(s.filename)

	if err != nil {
		fmt.Errorf("error opening file %s %v\n", s.filename, err)
		return nil
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	for {
		aud := <-s.audC

		w.Write(aud.buf)
	}

	return nil
}
