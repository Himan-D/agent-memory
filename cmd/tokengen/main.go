package main

import (
	"encoding/json"
	"fmt"
	"os"

	"agent-memory/internal/auth"
)

func main() {
	auth.InitAPIKeySalt("")
	bundle, err := auth.GenerateTokenBundle()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate tokens: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(bundle); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
		os.Exit(1)
	}
}
