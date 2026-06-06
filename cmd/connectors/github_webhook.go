package main

import "encoding/json"

func parseGitHubAction(body []byte) (string, bool) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	action, ok := payload["action"].(string)
	return action, ok
}
