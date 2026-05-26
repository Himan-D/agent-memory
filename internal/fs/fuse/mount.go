//go:build linux

package fuse

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"agent-memory/internal/fs/vfs"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type Filesystem struct {
	vfs    *vfs.VirtualFS
	mu     sync.RWMutex
	opened map[uint64]bool
}

func NewFilesystem(v *vfs.VirtualFS) *Filesystem {
	return &Filesystem{
		vfs:    v,
		opened: make(map[uint64]bool),
	}
}

func (f *Filesystem) Root() (fs.Node, error) {
	return &DirNode{fs: f, path: "/"}, nil
}

func Mount(mountPoint string, svc vfs.ServiceInterface) error {
	if svc == nil {
		return fmt.Errorf("service cannot be nil")
	}

	v := vfs.NewVirtualFS(svc, mountPoint)

	opts := []fuse.MountOption{
		fuse.FSName("agentfs"),
		fuse.AllowOther(),
	}

	c, err := fuse.Mount(mountPoint, opts...)
	if err != nil {
		return fmt.Errorf("fuse mount: %w", err)
	}
	defer c.Close()

	log.Printf("agentfs mounted at %s", mountPoint)

	fsys := NewFilesystem(v)
	err = fs.Serve(c, fsys)
	return err
}

func Unmount(mountPoint string) error {
	return fuse.Unmount(mountPoint)
}

type DirNode struct {
	fs   *Filesystem
	path string
}

func (n *DirNode) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o755
	return nil
}

func (n *DirNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fullPath := filepath.Join(n.path, name)

	_, inode, ok := n.fs.vfs.GetInodeByPath(fullPath)
	if !ok {
		return nil, syscall.ENOENT
	}

	if inode.IsDir {
		return &DirNode{fs: n.fs, path: fullPath}, nil
	}
	return &FileNode{fs: n.fs, path: fullPath}, nil
}

func (n *DirNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := n.fs.vfs.ReadDir(ctx, n.path)
	if err != nil {
		return nil, err
	}

	dirEntries := make([]fuse.Dirent, 0, len(entries))
	for _, entry := range entries {
		typ := fuse.DT_File
		if entry.IsDir {
			typ = fuse.DT_Dir
		}
		dirEntries = append(dirEntries, fuse.Dirent{
			Name: entry.Name,
			Type: typ,
		})
	}
	return dirEntries, nil
}

func (n *DirNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	fullPath := filepath.Join(n.path, req.Name)
	n.fs.vfs.AddDir(fullPath, req.Name)
	return &DirNode{fs: n.fs, path: fullPath}, nil
}

func (n *DirNode) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fullPath := filepath.Join(n.path, req.Name)

	node := &FileNode{fs: n.fs, path: fullPath}

	n.fs.mu.Lock()
	n.fs.opened[0] = true
	n.fs.mu.Unlock()

	return node, node, nil
}

type FileNode struct {
	fs   *Filesystem
	path string
}

func (n *FileNode) Attr(ctx context.Context, a *fuse.Attr) error {
	_, inode, ok := n.fs.vfs.GetInodeByPath(n.path)
	if !ok {
		a.Mode = 0o644
		return nil
	}
	a.Inode = inode.ID
	a.Mode = 0o644
	a.Size = inode.Size
	a.Mtime = inode.ModTime
	return nil
}

func (n *FileNode) ReadAll(ctx context.Context) ([]byte, error) {
	data, err := n.fs.vfs.ReadFile(ctx, n.path)
	if err != nil {
		return nil, syscall.EIO
	}
	return data, nil
}

func (n *FileNode) Write(ctx context.Context, req *fuse.WriteRequest) error {
	existing, _ := n.fs.vfs.ReadFile(ctx, n.path)
	newContent := append(existing, req.Data...)
	return n.fs.vfs.WriteFile(ctx, n.path, newContent)
}

func (n *FileNode) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return nil
}

func (n *FileNode) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return nil
}
