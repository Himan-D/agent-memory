package integration

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func waitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp != nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

// sendRESPPing sends a minimal RESP PING to a Redis server and expects +PONG reply
func sendRESPPing(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		defer conn.Close()
		// RESP for PING: *1\r\n$4\r\nPING\r\n
		req := "*1\r\n$4\r\nPING\r\n"
		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, err = conn.Write([]byte(req))
		if err != nil {
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		r := bufio.NewReader(conn)
		line, err := r.ReadString('\n')
		if err != nil {
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		if len(line) > 0 && (line == "+PONG\r\n" || line == "+PONG\n") {
			return nil
		}
		// unexpected response, retry
		conn.Close()
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for redis at %s", addr)
}

func TestExternalServicesAvailable(t *testing.T) {
	// Allow environment overrides for CI
	neoURL := os.Getenv("NEO4J_HTTP_URL")
	if neoURL == "" {
		neoURL = "http://localhost:7474/"
	}
	qdrURL := os.Getenv("QDRANT_URL")
	if qdrURL == "" {
		qdrURL = "http://localhost:6333/"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	t.Logf("Waiting for Neo4j at %s", neoURL)
	if err := waitForHTTP(neoURL, 90*time.Second); err != nil {
		t.Fatalf("neo4j not ready: %v", err)
	}

	t.Logf("Waiting for Qdrant at %s", qdrURL)
	if err := waitForHTTP(qdrURL, 90*time.Second); err != nil {
		t.Fatalf("qdrant not ready: %v", err)
	}

	t.Logf("Waiting for Redis at %s", redisAddr)
	if err := sendRESPPing(redisAddr, 90*time.Second); err != nil {
		t.Fatalf("redis not ready: %v", err)
	}

	t.Log("All external dev services are reachable")
}
