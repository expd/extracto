package main

import (
	"flag"
	"fmt"
	"github.com/expd/extracto/pkg/extract"
	"os"
)

func main() {
	var (
		source      = flag.String("source", "", "mpegts source address")
		tuner       = flag.String("tuner", "99999", "tuner id to set")
		targetVideo = flag.String("targetVideo", "", "video receiver address")
		targetAudio = flag.String("targetAudio", "", "streamer relay address")
	)
	flag.Parse()

	if len(*source) == 0 {
		fmt.Println("source cannot be empty")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Starting Extractor version %v\n", Version)

	err, ex := extract.CreateExtractor(source, tuner, targetVideo, targetAudio)

	err = ex.Init()
	defer ex.Close()

	if err != nil {
		panic(err)
	}

	err = ex.Extract()
	if err != nil {
		panic(err)
	}

}
