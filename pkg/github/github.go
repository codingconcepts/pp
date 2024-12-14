package github

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"pp/pkg/model"
	"runtime"
	"strings"

	"github.com/samber/lo"
	prog "github.com/schollz/progressbar/v3"
)

var (
	goos   = runtime.GOOS
	goarch = runtime.GOARCH

	osMap = map[string][]string{
		"darwin": {"macos"},
	}
)

type Client struct {
	httpClient *http.Client
	goos       string
	goarch     string
}

// NewClient returns a pointer to a new instance of Client.
func NewClient(httpClient *http.Client, opts ...Opt) *Client {
	c := Client{
		httpClient: httpClient,
		goos:       runtime.GOOS,
		goarch:     runtime.GOARCH,
	}

	for _, opt := range opts {
		opt(&c)
	}

	return &c
}

// ChooseBinary selects the most appropriate release asset based on the
// caller's OS and architecture.
func (c *Client) ChooseBinary(assets []model.Asset) (string, error) {
	assets = lo.Filter(assets, func(a model.Asset, _ int) bool {
		return strings.Contains(a.Name, c.goarch)
	})

	exactOS, ok := lo.Find(assets, func(a model.Asset) bool {
		return strings.Contains(a.Name, c.goos)
	})
	if ok {
		return exactOS.DownloadURL, nil
	}

	equivalentOS, ok := lo.Find(assets, func(a model.Asset) bool {
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

// GetLatestRelease fetches the latest version of a release, including the
// details of each of its assets.
func (c *Client) GetLatestRelease(owner, repo string) (*model.Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
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

	var release model.Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &release, nil
}

// DownloadFile downloads a given url and writes it to a given filePath.
func (c *Client) DownloadFile(url, filePath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 response code: %s", resp.Status)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	log.Println()
	bar := prog.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)

	if _, err = io.Copy(io.MultiWriter(out, bar), resp.Body); err != nil {
		return fmt.Errorf("writing file body: %w", err)
	}

	return nil
}
