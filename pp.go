package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
)

var (
	goos   = runtime.GOOS
	goarch = runtime.GOARCH

	osMap = map[string][]string{
		"darwin": {"macos"},
	}
)

type release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name          string `json:"name"`
	DownloadURL   string `json:"browser_download_url"`
	Size          int    `json:"size"`
	DownloadCount int    `json:"download_count"`
}

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out: os.Stdout,
		PartsExclude: []string{
			zerolog.TimestampFieldName,
		},
	}).Level(zerolog.DebugLevel)

	args := os.Args
	if len(args) != 3 {
		logger.Fatal().Msg("Usage: program <owner> <repo>")
	}

	owner, repo := args[1], args[2]

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	release, err := getLatestRelease(client, owner, repo)
	if err != nil {
		logger.Fatal().Err(err).Msg("error fetching latest release")
	}
	logger.Debug().Str("tag", release.TagName).Msg("latest release")

	for _, asset := range release.Assets {
		sizeMB := float64(asset.Size) / 1024 / 1024
		logger.Debug().
			Str("name", asset.Name).
			Str("size", fmt.Sprintf("%.2f MB", sizeMB)).
			Str("name", asset.Name).
			Str("name", asset.Name).
			Msg("")
	}

	binaryURL, err := chooseBinary(release.Assets)
	if err != nil {
		logger.Fatal().Err(err).Msg("error choosing binary")
	}

	fileName := filepath.Base(binaryURL)

	if err := downloadFile(client, binaryURL, fileName); err != nil {
		logger.Fatal().Err(err).Msg("error downloading binary")
	}

	if err = extractSingleFile(fileName, repo); err != nil {
		logger.Fatal().Err(err).Msg("error extracting binary")
	}

	if err = makeFileExecutable(repo); err != nil {
		logger.Fatal().Err(err).Msg("error making binary executable")
	}
}

func chooseBinary(assets []asset) (string, error) {
	assets = lo.Filter(assets, func(a asset, _ int) bool {
		return strings.Contains(a.Name, goarch)
	})

	exactOS, ok := lo.Find(assets, func(a asset) bool {
		return strings.Contains(a.Name, goos)
	})
	if ok {
		return exactOS.DownloadURL, nil
	}

	equivalentOS, ok := lo.Find(assets, func(a asset) bool {
		options, ok := osMap[goos]
		if !ok {
			return false
		}

		for _, opt := range options {
			if strings.Contains(a.Name, opt) {
				return true
			}
		}

		return false
	})
	if ok {
		return equivalentOS.DownloadURL, nil
	}

	return "", fmt.Errorf("no matching binaries found for os=%s arch=%s", goos, goarch)
}

func getLatestRelease(client *http.Client, owner, repo string) (*release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response code: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetching response: %w", err)
	}

	var release release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &release, nil
}

func downloadFile(client *http.Client, url, filepath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 response code: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("writing file body: %w", err)
	}

	return nil
}

func extractSingleFile(sourcePath, destName string) (err error) {
	defer func() {
		if cerr := os.Remove(sourcePath); err != nil {
			err = cerr
		}
	}()

	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("opening tar file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	if _, err = tarReader.Next(); err == io.EOF {
		return fmt.Errorf("archive is empty")
	}
	if err != nil {
		return fmt.Errorf("reading tar file header: %w", err)
	}

	out, err := os.OpenFile(destName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, tarReader)
	if err != nil {
		return fmt.Errorf("writing file body: %w", err)
	}

	if written == 0 {
		return fmt.Errorf("no data was written to the output file")
	}

	if err = out.Sync(); err != nil {
		return fmt.Errorf("syncing file to disk: %w", err)
	}

	return
}

func makeFileExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	// Add +x permissions.
	mode := info.Mode()
	executableMode := mode | 0111

	if err := os.Chmod(path, executableMode); err != nil {
		return fmt.Errorf("applying file permissions: %w", err)
	}

	return nil
}
