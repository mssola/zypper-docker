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
	"bytes"
	"errors"

	"github.com/SUSE/zypper-docker/backend/drivers"
)

// TODO: this package assumes that drivers are all CLI based, which is not
// necessarily true. We should use the `drivers.needsCLI` function and act
// accordingly.

// Initialize initializes the backend of zypper-docker. The parameter `server`
// indicates whether zypper-docker is running in server mode or not.
// TODO: hint backend
func Initialize(server bool) {
	// TODO
	if !server {
		listenSignals()
	}

	// TODO: available backends and so on
}

func isSupported(id string) bool {
	supported, _ := getCacheFile().isSupported(id)
	return supported
}

// CurrentDriver returns the driver to be used for the given image.
// TODO: change name to DriverForImage
func CurrentDriver(image string) drivers.Driver {
	// TODO: we have to sort this out. Sometimes the ID is given (because of
	// internal usage), but most of the times (since multiple driver support) it
	// comes from a CLI argument. This means that we don't have the image ID and
	// we would have to make another API call just for this *every* time. Since
	// this is expensive, my idea is to allow a mixture of <image:tag> and
	// <imageID>. In cache fails we will fetch the image object and store
	// <image:tag> and <imageID>. This, of course, comes in the expense of
	// having a larger cache size, but I don't think it will ever grow to a
	// point where performance harms us.
	return CurrentDriverWithID(image)
}

// TODO: implement an empty driver for unsupported stuff
func CurrentDriverWithID(id string) drivers.Driver {
	exists, driver := getCacheFile().isSupported(id)
	if !exists {
		return &drivers.Zypper{}
	}

	if val, ok := drivers.AvailableDrivers[driver]; ok {
		return val
	}
	return &drivers.Zypper{}
}

// SeveritySupported returns an error if there are some problems with the
// `--severity` flag for the given image.
func SeveritySupported(image string) error {
	current := CurrentDriver(image)
	cmd := current.SeverityCommand()
	if cmd == "" {
		return errors.New("the `--severity` flag is not supported for this image")
	}

	buf := bytes.NewBuffer([]byte{})
	id, err := runCommandInContainer(image, []string{cmd}, buf)
	if err != nil {
		return err
	}
	defer removeContainer(id)

	if !current.SeveritySupported(buf.String()) {
		return errors.New("the `--severity` flag is not supported for this image")
	}
	return nil
}
