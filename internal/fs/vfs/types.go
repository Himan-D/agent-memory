package vfs

import (
	"time"

	"agent-memory/internal/memory/types"
)

type Inode struct {
	ID       uint64
	Mode     uint32
	Size     uint64
	ModTime  time.Time
	IsDir    bool
	Name     string
	Path     string
	MemoryID string
	EntityID string
	Metadata map[string]string
}

type DirEntry struct {
	Name  string
	Inode uint64
	IsDir bool
}

type FileAttr struct {
	Ino   uint64
	Size  uint64
	Blocks uint64
	Atime uint64
	Mtime uint64
	Ctime uint64
	Mode  uint32
	Nlink uint32
	UID   uint32
	GID   uint32
	Rdev  uint32
}

type MemoryFile struct {
	Inode    *Inode
	Content  string
	Changed  bool
	Metadata *types.Memory
}

type Directory struct {
	Inode    *Inode
	Children map[string]*DirEntry
	Parent   *Directory
}
