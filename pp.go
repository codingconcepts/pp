package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"pp/pkg/fs"
	"pp/pkg/github"
	"time"

	"github.com/samber/lo"
)

func main() {
	log.SetFlags(0)

	args := os.Args
	if len(args) != 3 {
		log.Fatalln("usage: pp <owner> <repo>")
	}

	owner, repo := args[1], args[2]

	client := github.NewClient(&http.Client{Timeout: time.Second * 10})

	release, err := client.GetLatestRelease(owner, repo)
	if err != nil {
		log.Fatalf("error fetching latest release: %v", err)
	}
	log.Printf("latest version: %s", release.TagName)

	binaryURL, err := client.ChooseBinary(release.Assets)
	if err != nil {
		log.Fatalf("error choosing binary: %v", err)
	}

	log.Println("\nreleases:")
	for _, asset := range release.Assets {
		sizeMB := float64(asset.Size) / 1024 / 1024
		log.Printf("  - %s (%s)%s",
			asset.Name,
			fmt.Sprintf("%.2f MB", sizeMB),
			lo.Ternary(binaryURL == asset.DownloadURL, " *", ""),
		)
	}

	fileName := filepath.Base(binaryURL)

	if err := client.DownloadFile(binaryURL, fileName); err != nil {
		log.Fatalf("error downloading binary: %v", err)
	}

	if err = fs.ExtractBinary(fileName, repo); err != nil {
		log.Fatalf("error extracting binary: %v", err)
	}

	if err = fs.MakeExecutable(repo); err != nil {
		log.Fatalf("error making binary executable: %v", err)
	}
}
