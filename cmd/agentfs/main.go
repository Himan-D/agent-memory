// AgentFS — mount agent memory as a filesystem.
//
// Modes:
//
//	webdav  (default on macOS/Windows) — HTTP WebDAV server, mount via Finder/Explorer
//	fuse    (Linux) — native FUSE mount
//	local   — mirror VirtualFS into a local directory once (export)
//
// Examples:
//
//	agentfs -mode webdav -addr :8081 -api http://localhost:8080 -api-key $KEY
//	agentfs -mode local -mount ./agent-mem -api http://localhost:8080 -api-key $KEY
//	agentfs -mode fuse -mount /mnt/agent -api-key $KEY   # linux only
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"agent-memory/internal/fs/vfs"
	"agent-memory/internal/fs/webdav"
	"agent-memory/internal/storage"
)

func main() {
	defaultMode := "webdav"
	if runtime.GOOS == "linux" {
		defaultMode = "fuse"
	}

	mountPoint := flag.String("mount", defaultMount(), "Mount/export path (fuse/local)")
	apiURL := flag.String("api", envOr("AGENT_MEMORY_URL", "http://localhost:8080"), "Memory API base URL")
	apiKey := flag.String("api-key", os.Getenv("AGENT_MEMORY_API_KEY"), "API key (or AGENT_MEMORY_API_KEY)")
	tenantID := flag.String("tenant", envOr("AGENT_MEMORY_TENANT", "default"), "Tenant ID")
	mode := flag.String("mode", defaultMode, "Mode: webdav | fuse | local")
	addr := flag.String("addr", ":8081", "WebDAV listen address")
	storageProvider := flag.String("storage", envOr("STORAGE_PROVIDER", "local"), "Archive backend: local | s3 | gcs")
	dataDir := flag.String("data-dir", envOr("DATA_DIR", "./data/agentfs"), "Local blob data dir")
	s3Bucket := flag.String("s3-bucket", os.Getenv("S3_BUCKET"), "S3 bucket for archive/")
	gcsBucket := flag.String("gcs-bucket", os.Getenv("GCS_BUCKET"), "GCS bucket for archive/")
	awsRegion := flag.String("aws-region", envOr("AWS_REGION", "us-east-1"), "AWS region")
	status := flag.Bool("status", false, "Print status and exit (requires running against API)")
	umount := flag.Bool("umount", false, "Unmount FUSE mount point (linux)")
	flag.Parse()

	if *umount {
		if err := unmountFUSE(*mountPoint); err != nil {
			log.Fatalf("unmount: %v", err)
		}
		fmt.Printf("Unmounted %s\n", *mountPoint)
		return
	}

	if *apiKey == "" && *mode != "local" {
		// allow empty for status dry-run with null
		log.Println("warning: no API key set; set -api-key or AGENT_MEMORY_API_KEY")
	}

	svc := vfs.NewHTTPClient(*apiURL, *apiKey, *tenantID)

	var blob storage.BlobStore
	var err error
	switch strings.ToLower(*storageProvider) {
	case "s3":
		blob, err = storage.NewS3Store(*s3Bucket, *awsRegion)
	case "gcs":
		blob, err = storage.NewGCSStore(*gcsBucket)
	case "local", "":
		blob, err = storage.NewLocalStore(filepath.Join(*dataDir, "archive"))
	default:
		log.Fatalf("unknown storage provider: %s", *storageProvider)
	}
	if err != nil {
		log.Printf("warning: archive backend unavailable: %v (archive/ disabled)", err)
		blob = nil
	}

	v := vfs.NewVirtualFSWithOptions(svc, *mountPoint, *tenantID, blob, "agentfs/"+*tenantID+"/")

	if *status {
		fmt.Println(v.Status())
		fmt.Printf("api=%s tenant=%s mode=%s\n", *apiURL, *tenantID, *mode)
		// quick probe
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		entries, err := v.ReadDir(ctx, "/memories")
		if err != nil {
			fmt.Printf("memories: error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("memories: %d entries\n", len(entries))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down agentfs...")
		cancel()
	}()

	switch strings.ToLower(*mode) {
	case "webdav":
		srv := webdav.NewServer(v)
		if err := srv.Start(*addr); err != nil {
			log.Fatalf("webdav: %v", err)
		}
		fmt.Printf("agentfs WebDAV ready at http://localhost%s/\n", *addr)
		fmt.Println("macOS: Finder → Go → Connect to Server → http://localhost" + *addr + "/")
		fmt.Println("Tree: /memories  /skills  /sessions  /search  /archive")
		fmt.Println("Write /memories/new.md to create a memory. Ctrl+C to stop.")
		<-ctx.Done()
		_ = srv.Stop(context.Background())

	case "local":
		if err := exportLocal(ctx, v, *mountPoint); err != nil {
			log.Fatalf("local export: %v", err)
		}
		fmt.Printf("Exported agentfs tree to %s\n", *mountPoint)
		fmt.Println("Edit files then re-run with -mode local -sync (or use webdav for live writeback).")

	case "fuse":
		if err := mountFUSE(*mountPoint, svc, v); err != nil {
			log.Fatalf("fuse: %v", err)
		}
		fmt.Printf("agentfs FUSE mounted at %s\n", *mountPoint)
		<-ctx.Done()
		_ = unmountFUSE(*mountPoint)

	default:
		log.Fatalf("unknown mode %q (webdav|fuse|local)", *mode)
	}
}

func exportLocal(ctx context.Context, v *vfs.VirtualFS, root string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return walkExport(ctx, v, "/", root)
}

func walkExport(ctx context.Context, v *vfs.VirtualFS, virt, disk string) error {
	entries, err := v.ReadDir(ctx, virt)
	if err != nil {
		return err
	}
	for _, e := range entries {
		vp := filepath.ToSlash(filepath.Join(virt, e.Name))
		if !strings.HasPrefix(vp, "/") {
			vp = "/" + vp
		}
		dp := filepath.Join(disk, e.Name)
		if e.IsDir {
			if err := os.MkdirAll(dp, 0o755); err != nil {
				return err
			}
			if err := walkExport(ctx, v, vp, dp); err != nil {
				log.Printf("warn: %s: %v", vp, err)
			}
			continue
		}
		data, err := v.ReadFile(ctx, vp)
		if err != nil {
			log.Printf("warn: read %s: %v", vp, err)
			continue
		}
		if err := os.WriteFile(dp, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func defaultMount() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.TempDir(), "agentfs")
	}
	return "/mnt/agent"
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
