//go:build linux

package main

import (
	"agent-memory/internal/fs/fuse"
	"agent-memory/internal/fs/vfs"
)

func mountFUSE(mountPoint string, svc vfs.ServiceInterface, v *vfs.VirtualFS) error {
	_ = v
	return fuse.Mount(mountPoint, svc)
}

func unmountFUSE(mountPoint string) error {
	return fuse.Unmount(mountPoint)
}
