package thumbs

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// PackOffset stores the byte position and length of a thumbnail inside thumbs.pack
type PackOffset struct {
	Offset int64
	Length uint32
}

// LRUItem represents a cached thumbnail buffer
type LRUItem struct {
	photoID uint32
	data    []byte
	size    int
}

// PackStore manages single-file append-only thumbnail storage with in-memory offset index and 128MB LRU cache.
type PackStore struct {
	mu          sync.RWMutex
	packPath    string
	idxPath     string
	packFile    *os.File
	offsets     map[uint32]PackOffset
	lruList     *list.List
	lruMap      map[uint32]*list.Element
	currentSize int
	maxCacheSize int // default 128 MB
}

// NewPackStore opens or creates the thumbs.pack archive and thumbs.idx index.
func NewPackStore(dir string) (*PackStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	packPath := filepath.Join(dir, "thumbs.pack")
	idxPath := filepath.Join(dir, "thumbs.idx")

	packFile, err := os.OpenFile(packPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open thumbs.pack: %w", err)
	}

	ps := &PackStore{
		packPath:     packPath,
		idxPath:      idxPath,
		packFile:     packFile,
		offsets:      make(map[uint32]PackOffset, 10000),
		lruList:      list.New(),
		lruMap:       make(map[uint32]*list.Element),
		maxCacheSize: 128 * 1024 * 1024, // 128 MB
	}

	if err := ps.loadIndex(); err != nil {
		// Non-fatal if index empty or newly created
		_ = err
	}

	return ps, nil
}

// Close closes the underlying pack file.
func (ps *PackStore) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.packFile != nil {
		return ps.packFile.Close()
	}
	return nil
}

// loadIndex loads all offset entries from thumbs.idx into memory.
func (ps *PackStore) loadIndex() error {
	f, err := os.Open(ps.idxPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	// Each index entry: [photoID: 4 bytes][offset: 8 bytes][length: 4 bytes] = 16 bytes
	entryBuf := make([]byte, 16)
	for {
		_, err := io.ReadFull(f, entryBuf)
		if err != nil {
			break
		}
		id := binary.BigEndian.Uint32(entryBuf[0:4])
		off := int64(binary.BigEndian.Uint64(entryBuf[4:12]))
		lenBytes := binary.BigEndian.Uint32(entryBuf[12:16])
		ps.offsets[id] = PackOffset{
			Offset: off,
			Length: lenBytes,
		}
	}
	return nil
}

// Has checks if a thumbnail already exists in the archive for the given photo ID.
func (ps *PackStore) Has(photoID uint32) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	_, exists := ps.offsets[photoID]
	return exists
}

// Put writes a thumbnail into thumbs.pack and updates the index and LRU cache.
func (ps *PackStore) Put(photoID uint32, jpegData []byte) error {
	if len(jpegData) == 0 {
		return fmt.Errorf("empty thumbnail data")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Seek to end of pack file
	offset, err := ps.packFile.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek end of pack: %w", err)
	}

	length := uint32(len(jpegData))
	if _, err := ps.packFile.Write(jpegData); err != nil {
		return fmt.Errorf("failed to write thumb to pack: %w", err)
	}

	po := PackOffset{Offset: offset, Length: length}
	ps.offsets[photoID] = po

	// Append entry to thumbs.idx
	idxFile, err := os.OpenFile(ps.idxPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		entryBuf := make([]byte, 16)
		binary.BigEndian.PutUint32(entryBuf[0:4], photoID)
		binary.BigEndian.PutUint64(entryBuf[4:12], uint64(offset))
		binary.BigEndian.PutUint32(entryBuf[12:16], length)
		_, _ = idxFile.Write(entryBuf)
		_ = idxFile.Close()
	}

	// Update LRU cache
	ps.addToLRU(photoID, jpegData)

	return nil
}

// Get reads thumbnail bytes from LRU cache if available, or reads directly from thumbs.pack.
func (ps *PackStore) Get(photoID uint32) ([]byte, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// 1. Check LRU Cache
	if elem, ok := ps.lruMap[photoID]; ok {
		ps.lruList.MoveToFront(elem)
		item := elem.Value.(*LRUItem)
		return item.data, nil
	}

	// 2. Lookup offset
	po, exists := ps.offsets[photoID]
	if !exists {
		return nil, os.ErrNotExist
	}

	// 3. Read from thumbs.pack
	buf := make([]byte, po.Length)
	if _, err := ps.packFile.ReadAt(buf, po.Offset); err != nil {
		return nil, fmt.Errorf("failed to read thumb from pack: %w", err)
	}

	// 4. Add to LRU
	ps.addToLRU(photoID, buf)

	return buf, nil
}

func (ps *PackStore) addToLRU(photoID uint32, data []byte) {
	// If already in LRU, update
	if elem, ok := ps.lruMap[photoID]; ok {
		ps.lruList.MoveToFront(elem)
		item := elem.Value.(*LRUItem)
		ps.currentSize -= item.size
		item.data = data
		item.size = len(data)
		ps.currentSize += item.size
		return
	}

	item := &LRUItem{
		photoID: photoID,
		data:    data,
		size:    len(data),
	}
	elem := ps.lruList.PushFront(item)
	ps.lruMap[photoID] = elem
	ps.currentSize += len(data)

	// Evict oldest until within limit
	for ps.currentSize > ps.maxCacheSize && ps.lruList.Len() > 0 {
		oldest := ps.lruList.Back()
		if oldest == nil {
			break
		}
		oldItem := oldest.Value.(*LRUItem)
		ps.currentSize -= oldItem.size
		delete(ps.lruMap, oldItem.photoID)
		ps.lruList.Remove(oldest)
	}
}
