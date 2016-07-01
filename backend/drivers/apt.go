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

import "fmt"

type Apt struct{}

func (apt *Apt) GeneralUpdate() (string, error) {
	return apt.SecurityUpdate()
}

func (apt *Apt) SecurityUpdate() (string, error) {
	flags := "-y -V"
	return fmt.Sprintf("apt-get -qq -y update && apt-get %s upgrade "+
		"&& apt-get clean -qq -y", flags), nil
}

func (apt *Apt) ListGeneralUpdates() (string, error) {
	return apt.ListSecurityUpdates(false)
}

func (apt *Apt) ListSecurityUpdates(machine bool) (string, error) {
	flags := "--just-print"

	if machine {
		flags += " -q"
	} else {
		flags += " -V"
	}
	return fmt.Sprintf("apt-get -y update && apt-get %s upgrade", flags), nil
}

// TODO
func (apt *Apt) ParseUpdateOutput(output []byte) Updates {
	return Updates{}
}

// TODO
func (apt *Apt) CheckPatches() (string, error) {
	return "", nil
}

func (apt *Apt) IsExitCodeSevere(code int) (bool, error) {
	return code != 0, nil
}

func (apt *Apt) NeedsCLI() bool {
	return true
}

func (apt *Apt) Available() string {
	return "apt-get"
}

func (apt *Apt) SeverityCommand() string {
	return ""
}

func (apt *Apt) SeveritySupported(output string) bool {
	return false
}
