package vfs

import (
	"sync"
	"time"
)

// InodeManager manages inode allocation and lookup
// Follows the session pool pattern from neo4j/client.go
type InodeManager struct {
	mu        sync.RWMutex
	inodes    map[uint64]*Inode
	pathToID  map[string]uint64
	nextInode uint64
	freeList  chan uint64  // Pool of reusable inode numbers
	maxInodes uint64
}

// NewInodeManager creates a new inode manager
// Pattern: like NewClient() in neo4j/client.go
func NewInodeManager(maxInodes uint64) *InodeManager {
	if maxInodes == 0 {
		maxInodes = 1000000  // Default 1M inodes
	}

	return &InodeManager{
		inodes:   make(map[uint64]*Inode),
		pathToID: make(map[string]uint64),
		nextInode: 1,  // 0 is reserved for root
		freeList:  make(chan uint64, 1000),  // Buffer pool
		maxInodes: maxInodes,
	}
}

// Allocate creates a new inode for a file or directory
// Returns existing inode if path already exists
func (m *InodeManager) Allocate(path, name string, isDir bool, memoryID, entityID string) (uint64, *Inode) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullPath := path
	if path != "" && path != "/" && name != "" {
		fullPath = path + "/" + name
	} else if name != "" {
		fullPath = "/" + name
	}

	// Check if path already exists
	if id, ok := m.pathToID[fullPath]; ok {
		if inode, ok := m.inodes[id]; ok {
			return id, inode
		}
	}

	// Try to reuse from free list (pool pattern)
	var id uint64
	select {
	case id = <-m.freeList:
		// Reuse freed inode
	default:
		if m.nextInode >= m.maxInodes {
			return 0, nil  // No more inodes
		}
		id = m.nextInode
		m.nextInode++
	}

	now := time.Now()
	inode := &Inode{
		ID:        id,
		Mode:      getMode(isDir),
		Size:      0,
		ModTime:   now,
		IsDir:     isDir,
		Name:      name,
		Path:      fullPath,
		MemoryID:  memoryID,
		EntityID:  entityID,
		Metadata:  make(map[string]string),
	}

	m.inodes[id] = inode
	m.pathToID[fullPath] = id

	return id, inode
}

// GetByID retrieves an inode by its ID
func (m *InodeManager) GetByID(id uint64) (*Inode, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inode, ok := m.inodes[id]
	return inode, ok
}

// GetByPath retrieves an inode by its path
func (m *InodeManager) GetByPath(path string) (uint64, *Inode, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if id, ok := m.pathToID[path]; ok {
		if inode, ok := m.inodes[id]; ok {
			return id, inode, true
		}
	}
	return 0, nil, false
}

// UpdateSize updates the size of an inode
func (m *InodeManager) UpdateSize(id uint64, size uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inode, ok := m.inodes[id]; ok {
		inode.Size = size
		inode.ModTime = time.Now()
		return true
	}
	return false
}

// UpdateMode updates the mode of an inode
func (m *InodeManager) UpdateMode(id uint64, mode uint32) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inode, ok := m.inodes[id]; ok {
		inode.Mode = mode
		inode.ModTime = time.Now()
		return true
	}
	return false
}

// SetMetadata sets metadata on an inode
func (m *InodeManager) SetMetadata(id uint64, key, value string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inode, ok := m.inodes[id]; ok {
		inode.Metadata[key] = value
		inode.ModTime = time.Now()
		return true
	}
	return false
}

// Delete removes an inode (returns it to pool)
func (m *InodeManager) Delete(id uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inode, ok := m.inodes[id]; ok {
		delete(m.pathToID, inode.Path)
		delete(m.inodes, id)

		// Return to free list (pool pattern)
		select {
		case m.freeList <- id:
			// Returned to pool
		default:
			// Pool full, discard
		}
		return true
	}
	return false
}

// DeleteByPath removes an inode by path
func (m *InodeManager) DeleteByPath(path string) bool {
	if id, _, ok := m.GetByPath(path); ok {
		return m.Delete(id)
	}
	return false
}

// List returns all inodes (for debugging)
func (m *InodeManager) List() []*Inode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inodes := make([]*Inode, 0, len(m.inodes))
	for _, inode := range m.inodes {
		inodes = append(inodes, inode)
	}
	return inodes
}

// Count returns the number of allocated inodes
func (m *InodeManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.inodes)
}

// FreeCount returns the number of inodes in the free list
func (m *InodeManager) FreeCount() int {
	return len(m.freeList)
}

// Close cleans up the inode manager
func (m *InodeManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	close(m.freeList)
	m.inodes = make(map[uint64]*Inode)
	m.pathToID = make(map[string]uint64)
}

// Helper: getMode returns the appropriate mode for a file or directory
func getMode(isDir bool) uint32 {
	if isDir {
		return 0o755 | 0o40000  // Directory with rwxr-xr-x
	}
	return 0o644  // Regular file with rw-r--r--
}
