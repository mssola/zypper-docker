// Copyright (c) 2015 SUSE LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SUSE/zypper-docker/backend/drivers"
	"github.com/SUSE/zypper-docker/logger"
	"github.com/SUSE/zypper-docker/utils"
	"github.com/coreos/etcd/pkg/fileutil"
)

const (
	cacheName = "zypper-docker.json"
	otherKey  = "others"
)

// The representation of cached data for this application.
type cachedData struct {
	// The path to the original cache file.
	Path string `json:"-"`

	// Map containing the inspected images. It contains both images that can be
	// handled, and images that can't. Images that can be handled are stored in
	// their respective handler name (e.g. zypper), while unknown images
	// will be stored inside of the `otherKey` key.
	IDs map[string][]string `json:"ids"`

	// Contains all the IDs of the images that have been either patched or
	// upgraded or patched using zypper-docker
	Outdated []string `json:"outdated"`

	// Whether this data comes from a valid file or not.
	Valid bool `json:"-"`
}

// Checks whether the given Id exists or not. It returns two booleans:
//  - Whether it exists or not.
//  - If it exists, the name of the driver being used.
func (cd *cachedData) idExists(id string) (bool, string) {
	for key, ids := range cd.IDs {
		if utils.ArrayIncludeString(ids, id) {
			return true, key
		}
	}

	return false, ""
}

// Returns whether the given ID matches an image that has been
// updated via zypper-docker patch|update
func (cd *cachedData) isImageOutdated(id string) bool {
	return utils.ArrayIncludeString(cd.Outdated, id)
}

// Returns whether the given ID matches an image that is based on an image that
// is supported. If it is supported, it will also return the name of the driver
// being used.
func (cd *cachedData) isSupported(id string) (bool, string) {
	if cd.Valid {
		if exists, driver := cd.idExists(id); exists {
			return driver != otherKey, driver
		}
	}

	if cd.Valid {
		for k, driver := range drivers.AvailableDrivers {
			// TODO: hide output and improve it for debug mode
			if checkCommandInImage(id, driver.Available()) {
				cd.IDs[k] = append(cd.IDs[k], id)
				return true, k
			}
		}
		cd.IDs[otherKey] = append(cd.IDs[otherKey], id)
	}
	return false, ""
}

// Writes all the cached data back to the cache file. This is needed because
// functions like `inSUSE` only write to memory. Therefore, once you're done
// with this instance, you should call this function to keep everything synced.
func (cd *cachedData) flush() {
	if !cd.Valid {
		// Silently fail, the user has probably already been notified about it.
		return
	}

	file, err := fileutil.LockFile(cd.Path, os.O_RDWR, 0666)
	if err != nil {
		logger.Printf("cannot write to the cache file: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Stat()
	if err != nil {
		logger.Printf("cannot stat file: %v", err)
		return
	}

	// Read cache from file (again) before writing to it, otherwise the
	// cache will possibly be inconsistent.
	oldCache := cd.readCache(file)

	// Merge the old and "new" cache, and remove duplicates.
	for _, driver := range drivers.Available {
		cd.IDs[driver] = append(cd.IDs[driver], oldCache.IDs[driver]...)
		cd.IDs[driver] = utils.RemoveDuplicates(cd.IDs[driver])
	}
	cd.IDs[otherKey] = append(cd.IDs[otherKey], oldCache.IDs[otherKey]...)
	cd.IDs[otherKey] = utils.RemoveDuplicates(cd.IDs[otherKey])
	cd.Outdated = append(cd.Outdated, oldCache.Outdated...)
	cd.Outdated = utils.RemoveDuplicates(cd.Outdated)

	// Clear file content.
	file.Seek(0, 0)
	file.Truncate(0)

	enc := json.NewEncoder(file)
	_ = enc.Encode(cd)
}

// Empty the contents of the cache file.
func (cd *cachedData) reset() {
	cd.IDs = make(map[string][]string)
	cd.Outdated = []string{}
	cd.flush()
}

// Update the Cachefile after an update.
// The ID of the outdated image will be added to outdated Images and
// the ID of the new image will be added to the SUSE Images.
func (cd *cachedData) updateCacheAfterUpdate(outdatedImg, updatedImgID, backend string) error {
	outdatedImgID, err := getImageID(outdatedImg)
	if err != nil {
		return err
	}
	if !utils.ArrayIncludeString(cd.Outdated, outdatedImgID) {
		cd.Outdated = append(cd.Outdated, outdatedImgID)
		cd.flush()
	}
	if !utils.ArrayIncludeString(cd.IDs[backend], updatedImgID) {
		cd.IDs[backend] = append(cd.IDs[backend], updatedImgID)
		cd.flush()
	}

	return nil
}

func (cd *cachedData) readCache(r io.Reader) *cachedData {
	ret := &cachedData{Valid: true, Path: cd.Path}
	dec := json.NewDecoder(r)
	if err := dec.Decode(&ret); err != nil && err != io.EOF {
		logger.Printf("decoding of cache file failed: %v", err)
		return &cachedData{Valid: true, Path: cd.Path}
	}

	return ret
}

// Retrieves the path for the cache file. It checks the following directories
// in this specific order:
//  1. $HOME/.cache
//  2. /tmp
// It will try to open (or create if it doesn't exist) the cache file in each
// directory until it finds a directory that is accessible.
func cachePath() *os.File {
	candidates := []string{filepath.Join(os.Getenv("HOME"), ".cache"), "/tmp"}

	for _, dir := range candidates {
		dirs := strings.Split(dir, ":")
		for _, d := range dirs {
			name := filepath.Join(d, cacheName)
			lock, err := fileutil.LockFile(name, os.O_RDWR|os.O_CREATE, 0666)
			if err == nil {
				return lock.File
			}
		}
	}
	return nil
}

// Create a cache file or get the current one if it already exists. If that is
// not possible, then the returned struct will be marked as invalid (meaning
// that `isSUSE` will work without caching).
func getCacheFile() *cachedData {
	file := cachePath()
	if file == nil {
		logger.Printf("could not find path for the cache")
		return &cachedData{Valid: false}
	}

	cd := &cachedData{
		Valid: true,
		IDs:   make(map[string][]string),
		Path:  file.Name(),
	}
	dec := json.NewDecoder(file)
	err := dec.Decode(&cd)
	_ = file.Close()
	if err != nil && err != io.EOF {
		logger.Printf("decoding of cache file failed: %v", err)
		return &cachedData{Valid: true, Path: file.Name()}
	}
	return cd
}
