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
	"errors"
	"fmt"
)

type Dnf struct{}

const (
	dnfExitOK           = 0
	dnfExitErr          = 1
	dnfExitUpdateNeeded = 100
)

func (dnf *Dnf) GeneralUpdate() (string, error) {
	return dnf.SecurityUpdate()
}

func (dnf *Dnf) SecurityUpdate() (string, error) {
	flags := "--allowerasing --best -v -y --refresh"
	return fmt.Sprintf("dnf %s upgrade && dnf -q -y clean all", flags), nil
}

func (dnf *Dnf) ListGeneralUpdates() (string, error) {
	return dnf.ListSecurityUpdates(false)
}

func (dnf *Dnf) ListSecurityUpdates(machine bool) (string, error) {
	flags := "--allowerasing --best -y --refresh"

	if machine {
		flags += " -q"
	} else {
		flags += " -v"
	}
	return fmt.Sprintf("dnf %s check-update", flags), nil
}

// TODO
func (dnf *Dnf) ParseUpdateOutput(output []byte) Updates {
	return Updates{}
}

// TODO
func (dnf *Dnf) CheckPatches() (string, error) {
	return "", nil
}

func (dnf *Dnf) IsExitCodeSevere(code int) (bool, error) {
	switch code {
	case dnfExitErr:
		return true, errors.New("there was an error in dnf")
	case dnfExitUpdateNeeded:
		return true, nil
	}
	return false, nil
}

func (dnf *Dnf) NeedsCLI() bool {
	return true
}

func (dnf *Dnf) Available() string {
	return "dnf -h"
}

func (dnf *Dnf) SeverityCommand() string {
	return ""
}

func (dnf *Dnf) SeveritySupported(output string) bool {
	return false
}
