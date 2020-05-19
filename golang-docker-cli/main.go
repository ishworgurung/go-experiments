package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	if len(os.Args) == 1 {
		panic(errors.New("need image tag name as the first arg"))
	}
	imageName := os.Args[1]

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	var dockerFile string
	var dockerBaseImage string

	// image search
	foundImage, foundImageID := func(imageName string) (string, string) {
		imageList, err := cli.ImageList(context.Background(), types.ImageListOptions{})
		if err != nil {
			panic(err)
		}
		iName := ""
		iID := ""
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

	if len(foundImage) == 0 && len(foundImageID) == 0 {
		panic(errors.New("no such image found"))
	} else {
		dockerBaseImage += fmt.Sprintf("Found image '%s' with id '%s' ", foundImage, foundImageID)
	}

	// Dockerfile reconstruction (partial WIP)
	imageHistory, err := cli.ImageHistory(context.Background(), foundImageID)
	if err != nil {
		panic(err)
	}

	for _, ih := range imageHistory {
		if len(ih.Tags) == 0 {
			continue
		}
		// The image tag value [0] is the same as what we are looking for, skip.
		if ih.Tags[0] == foundImage {
			continue
		}
		dockerBaseImage += fmt.Sprintf(" built from base image '%s'\n", ih.Tags[0])
		dockerFile += fmt.Sprintf("FROM %s\n", ih.Tags[0])
		break
	}

	fmt.Println("------------------------------------------------------")
	fmt.Println(dockerBaseImage)
	fmt.Println("------------------------------------------------------")

	var l []string
	for i := len(imageHistory) - 1; i >= 0; i-- {
		history := imageHistory[i]
		cmd := strings.ReplaceAll(history.CreatedBy, "/bin/sh -c #(nop) ", "")
		cmd = strings.ReplaceAll(cmd, "/bin/sh -c", "RUN /bin/sh -c")
		cmd = strings.ReplaceAll(cmd, " CMD [\"/bin/sh\"]", "")
		cmd = strings.ReplaceAll(cmd, " CMD [\"/bin/bash\"]", "")
		cmd = strings.ReplaceAll(cmd, " ENV", "ENV")
		l = append(l, cmd)
	}

	for _, e := range l {
		if len(e) == 0 {
			continue
		}
		dockerFile += fmt.Sprintf("%s\n", e)
	}
	fmt.Println("------------------------------------------------------")
	fmt.Println("Complete Dockerfile")
	fmt.Println("------------------------------------------------------")
	fmt.Printf("%s", dockerFile)
}
