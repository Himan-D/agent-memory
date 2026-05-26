package extractor

import "os"

// lookupEnv returns the environment variable value or empty string.
func lookupEnv(k string) string {
	v, _ := os.LookupEnv(k)
	return v
}
