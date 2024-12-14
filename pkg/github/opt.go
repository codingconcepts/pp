package github

type Opt func(*Client)

// WithGOOS allows runtime.GOOS to be override for testing purposes.
func WithGOOS(goos string) Opt {
	return func(c *Client) {
		c.goos = goos
	}
}

// WithGOARCH allows runtime.GOARCH to be override for testing purposes.
func WithGOARCH(goarch string) Opt {
	return func(c *Client) {
		c.goarch = goarch
	}
}
