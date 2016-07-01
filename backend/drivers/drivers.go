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

package drivers

import (
	"github.com/codegangsta/cli"
	"github.com/docker/engine-api/types/container"
)

// Available contains a slice of strings identifying the available drivers.
var Available = []string{"apt", "zypper", "dnf"}

// TODO: replace Available above with this
var AvailableDrivers = map[string]Driver{
	"apt":    &Apt{},
	"dnf":    &Dnf{},
	"zypper": &Zypper{},
}

// UpdateInfo contains all the information regarding updates so it can be
// consumed later instead of being displayed right away.
/*
type UpdateInfo struct {
	Updates  int
	Security int
}
*/

// Driver is the interface that any backend has to provide.
type Driver interface {
	GeneralUpdate() (string, error)
	SecurityUpdate() (string, error)
	ListGeneralUpdates() (string, error)
	ListSecurityUpdates(machine bool) (string, error)
	// TODO
	ParseUpdateOutput(output []byte) Updates
	CheckPatches() (string, error)
	IsExitCodeSevere(code int) (bool, error)
	NeedsCLI() bool
	SeverityCommand() string
	SeveritySupported(output string) bool
	Available() string
}

// cliContext represents the context as given by the CLI.
var cliContext *cli.Context

// Initialize initializes the CLI component for drivers.
func Initialize(ctx *cli.Context) {
	cliContext = ctx
}

// GetHostConfig fetches the host config to be used for starting containers
// from the CLI option.
func GetHostConfig() *container.HostConfig {
	if cliContext == nil {
		return &container.HostConfig{}
	}
	return &container.HostConfig{
		ExtraHosts: cliContext.GlobalStringSlice("add-host"),
	}
}

type notSupportedError struct {
	name string
}

func (err notSupportedError) Error() string {
	return "action not supported by '" + err.name + "'"
}

// IsNotSupported returns whether the given error is of a driver complaining
// that an action was not supported.
func IsNotSupported(err error) bool {
	_, ok := err.(notSupportedError)
	return ok
}
