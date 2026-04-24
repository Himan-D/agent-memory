package connectors

type SlackClient struct {
	token      string
	signingSec string
}

type SlackMessage struct {
	Channel  string `json:"channel"`
	Content  string `json:"content"`
	ThreadTS string `json:"thread_ts,omitempty"`
	User     string `json:"user,omitempty"`
	TS       string `json:"ts"`
}

func NewSlackClient(token, signing string) *SlackClient {
	return &SlackClient{token: token}
}

func (c *SlackClient) PostMessage(channel, text string) *SlackMessage {
	return &SlackMessage{Channel: channel, Content: text}
}