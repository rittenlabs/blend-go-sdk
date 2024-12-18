/*

Copyright (c) 2022 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package profanity

// Logger are the methods required on the logger.
type Logger interface {
	Printf(string, ...interface{})
	Errorf(string, ...interface{})
}
