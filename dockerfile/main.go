package main

// Generate Dockerfile from a Docker image
import (
	"flag"
	"fmt"
)

func main() {
	imageIdOpt := flag.String("i", "", "-i [imageid|layerid]")
	imageNameOpt := flag.String("n", "", "-n [foobar:latest|foobar:1.1.2]")
	imageRepo := flag.String("r", "docker.io/library", "-r [0234551.dkr.ap-southeast-2.aws.com/foobar|docker.io/library]")
	flag.Parse()

	var (
		err error
	)
	// Either image id or image name tag must be provided.
	// The Image repo is optional and defaults to `docker.io/library/`
	if len(*imageNameOpt) == 0 && len(*imageIdOpt) == 0 {
		println("either image name or image id should be provided")
		flag.Usage()
	}

	dir := newDockerImageClient(*imageRepo)
	if len(*imageNameOpt) > 0 {
		dir.imageName = *imageNameOpt
		// Search the user provided image name to get the image id
		dir.imageId, err = dir.getImageIdByName(*imageNameOpt)
		if err != nil {
			dir.zlog.Warn().Msg(err.Error())
		}
		// Pull image from registry since it does not exist locally
		if len(dir.imageId) == 0 {
			dir.zlog.Warn().Msg("the image could not be found locally")
			if err = dir.pullImage(); err != nil {
				dir.zlog.Fatal().Msg(err.Error())
			}
		}
	} else {
		// Search the user provided image id to get the image name
		// Image pull does not happen here
		dir.imageName, err = dir.getBaseImageTagByImageId(*imageIdOpt)
		if err != nil {
			dir.zlog.Error().Msg(err.Error())
		}
		if len(dir.imageName) == 0 {
			dir.zlog.Fatal().Msg("the image could not be found locally")
		}
		dir.imageId = *imageIdOpt
	}

	dir.imageId, err = dir.getImageIdByName(dir.imageName)
	if err != nil {
		dir.zlog.Fatal().Msg(err.Error())
	}
	// Dockerfile reconstruction
	bi, err := dir.getBaseImageTagByImageId(dir.imageId)
	if err != nil {
		dir.zlog.Fatal().Msg(err.Error())
	}
	dir.dockerfile, err = dir.dockerFile(bi)
	if err != nil {
		dir.zlog.Fatal().Msg(err.Error())
	}
	fmt.Printf("%s", dir.dockerfile)
}
