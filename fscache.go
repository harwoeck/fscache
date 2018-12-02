package fscache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// FsCache holds the caching structure and needed internals to cache file-
// system content.
type FsCache struct {
	isprod bool
	mutex  sync.RWMutex
	memory map[string][]byte
}

// NewFsCache returns a new instance of FsCache
func NewFsCache(isProduction bool) *FsCache {
	return &FsCache{
		isprod: isProduction,
		memory: make(map[string][]byte),
	}
}

// Load looks if the file is already in the internal memory-buffer cache. If so
// it returns it immediately, else the file is loaded, stored to the cache and
// returned.
// If the FsCache is in production mode the file is loaded from disk each time
// it's requested. This feature can be used if the underlying file-content
// changes during development.
func (fs *FsCache) Load(path string) (buf []byte, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(path)
	if err != nil {
		return nil, err
	}

	// If we aren't in production just read the file every time from the file system
	// hence we don't need to restart GO-APP for updated content files
	if !fs.isprod {
		buf, err = ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	// Lookup if we have file already in memory
	fs.mutex.RLock()
	buf, exists := fs.memory[path]
	fs.mutex.RUnlock()

	// If not read it into memory and save in map
	if !exists {
		fileBuf, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		fs.mutex.Lock()
		fs.memory[path] = fileBuf
		fs.mutex.Unlock()

		return fileBuf, nil
	}

	// File already exists -> return it
	return buf, nil
}

func (fs *FsCache) preloadUnprotected(path string) (err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		return err
	}

	fileBuf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	fs.memory[path] = fileBuf
	return nil
}

// Preload loads the file in path, it it exists, into the internal caching
// structure
func (fs *FsCache) Preload(path string) (err error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	return fs.preloadUnprotected(path)
}

// PreloadBatch is like Preload but performs the operation for a full batch,
// while still only acquiring internal mutexes once.
func (fs *FsCache) PreloadBatch(paths []string) (err error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, p := range paths {
		err = fs.preloadUnprotected(p)
		if err != nil {
			return err
		}
	}

	return nil
}
