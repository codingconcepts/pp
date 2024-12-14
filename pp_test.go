package main

import (
	"net/http"
	"pp/pkg/github"
	"pp/pkg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChooseBinary(t *testing.T) {
	cases := []struct {
		name   string
		goos   string
		goarch string
		expURL string
		expErr error
	}{
		{
			name:   "darwin amd64",
			goos:   "darwin",
			goarch: "amd64",
			expURL: "a",
		},
		{
			name:   "macos arm64",
			goos:   "macos",
			goarch: "arm64",
			expURL: "b",
		},
		{
			name:   "darwin arm64",
			goos:   "darwin",
			goarch: "arm64",
			expURL: "c",
		},
		{
			name:   "linux amd64",
			goos:   "linux",
			goarch: "amd64",
			expURL: "d",
		},
		{
			name:   "windows amd64",
			goos:   "windows",
			goarch: "amd64",
			expURL: "e",
		},
	}

	assets := []model.Asset{
		{Name: "pp_v1.0.0_macos_amd64.tar.gz", DownloadURL: "a"},
		{Name: "pp_v1.0.0_macos_arm64.tar.gz", DownloadURL: "b"},
		{Name: "pp_v1.0.0_darwin_arm64.tar.gz", DownloadURL: "c"},
		{Name: "pp_v1.0.0_linux_amd64.tar.gz", DownloadURL: "d"},
		{Name: "pp_v1.0.0_windows_amd64.tar.gz", DownloadURL: "e"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Override GOOS and GOARCH.
			client := github.NewClient(http.DefaultClient, github.WithGOARCH(c.goarch), github.WithGOOS(c.goos))

			actURL, actErr := client.ChooseBinary(assets)
			assert.Equal(t, c.expErr, actErr)
			if actErr != nil {
				return
			}

			assert.Equal(t, c.expURL, actURL)
		})
	}
}
