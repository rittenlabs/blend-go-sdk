/*

Copyright (c) 2022 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package sh

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// PutOutput runs a given command with a given reader as its stdin and captures the output.
func PutOutput(stdin io.Reader, command string, args ...string) ([]byte, error) {
	absoluteCommand, err := exec.LookPath(command)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(absoluteCommand, args...)
	cmd.Env = os.Environ()
	buffer := new(bytes.Buffer)
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	cmd.Stdin = stdin
	return buffer.Bytes(), cmd.Run()
}
