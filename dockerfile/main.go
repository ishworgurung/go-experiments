package main

// Generate Dockerfile from a Docker image
import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
)

func main() {
	var (
		curr, baseImage             string
		dockerBaseImage, dockerFile string
		// good enough..
		reserved = []string{
			"ENV",
			"EXPOSE",
			"ARG",
			"LABEL",
			"USER",
			"EXPOSE",
			"CMD",
			"MAINTAINER",
			"ENTRYPOINT",
			"STOPSIGNAL",
		}
		replacer = strings.NewReplacer(
			"/bin/sh -c #(nop) ", "",
			"/bin/sh -c", "RUN /bin/sh -c",
			"&&", "\\\n    &&",
		)
	)

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log := zerolog.New(output).With().Timestamp().Logger()

	if len(os.Args) == 1 {
		log.Fatal().Msg("need image tag name as the first arg")
	}
	imageName := os.Args[1]

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal().Err(err)
	}

	// List images - search the user provided image
	originalImageName, originalImageID := func(imageName string) (string, string) {
		// Get image list
		imageList, err := cli.ImageList(context.Background(), types.ImageListOptions{})
		if err != nil {
			log.Fatal().Err(err)
		}
		var (
			iName, iID string
		)
		for _, image := range imageList {
			for _, i := range image.RepoTags {
				if len(i) > 0 && i == imageName {
					iName = i
					iID = image.ID
				}
			}
		}
		return iName, iID
	}(imageName)

	if len(originalImageName) == 0 && len(originalImageID) == 0 {
		log.Fatal().Msg("no such image found")
	}

	dockerBaseImage = fmt.Sprintf("I found the image '%s' with id '%s' ",
		originalImageName, originalImageID)

	// Dockerfile reconstruction - get image history.
	imageHistory, err := cli.ImageHistory(context.Background(), originalImageID)
	if err != nil {
		log.Fatal().Err(err)
	}
	for _, ih := range imageHistory {
		if len(ih.Tags) == 0 {
			continue
		}
		curr = ih.Tags[0]
	}
	if len(curr) == 0 {
		log.Fatal().Msg("The base image could not be found!")
	}

	if curr == originalImageName {
		baseImage = originalImageName
	} else {
		baseImage = curr
	}
	dockerBaseImage += fmt.Sprintf(" which was built from the base image '%s'", baseImage)
	log.Info().Msg(dockerBaseImage)

	dockerFile += fmt.Sprintf("FROM %s\n", baseImage)
	// traverse image history slice backwards
	for i := len(imageHistory) - 1; i >= 0; i-- {
		history := imageHistory[i].CreatedBy
		if len(history) == 0 {
			continue
		}
		steps := replacer.Replace(history)
		for _, e := range reserved {
			steps = strings.ReplaceAll(steps, " "+e, e)
		}
		dockerFile += fmt.Sprintf("%s\n", steps)
	}
	log.Info().Msgf("Complete Dockerfile with multi-stage build steps\n%s", dockerFile)
}
