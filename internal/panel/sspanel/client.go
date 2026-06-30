package sspanel

type Client struct {
	PanelURL string
	Token    string
}

func NewClient(panelURL string, token string) *Client {
	return &Client{
		PanelURL: panelURL,
		Token:    token,
	}
}
