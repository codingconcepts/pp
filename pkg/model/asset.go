package model

// Asset describes a file added to a release.
type Asset struct {
	Name          string `json:"name"`
	DownloadURL   string `json:"browser_download_url"`
	Size          int    `json:"size"`
	DownloadCount int    `json:"download_count"`
}
