package connectors

type GoogleDriveClient struct {
	clientID   string
	accessToken string
}

type GFile struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	MimeType string `json:"mimeType"`
	Link  string `json:"link"`
}

func NewGoogleDriveClient(clientID, accessToken string) *GoogleDriveClient {
	return &GoogleDriveClient{clientID: clientID}
}

func (c *GoogleDriveClient) ListFiles() []GFile {
	return []GFile{}
}