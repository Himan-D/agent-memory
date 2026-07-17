//go:build !linux

package main

import (
	"fmt"
	"runtime"

	"agent-memory/internal/fs/vfs"
)

func mountFUSE(mountPoint string, svc vfs.ServiceInterface, v *vfs.VirtualFS) error {
	_ = mountPoint
	_ = svc
	_ = v
	return fmt.Errorf("FUSE mode is only supported on Linux; use -mode webdav or -mode local on %s", runtime.GOOS)
}

func unmountFUSE(mountPoint string) error {
	_ = mountPoint
	return fmt.Errorf("FUSE unmount only supported on Linux")
}
