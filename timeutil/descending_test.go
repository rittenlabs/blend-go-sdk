/*

Copyright (c) 2022 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package timeutil

import (
	"sort"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestDescending(t *testing.T) {
	assert := assert.New(t)

	times := []time.Time{
		time.Date(2019, 02, 10, 00, 00, 00, 00, time.UTC),
		time.Date(2019, 02, 11, 00, 00, 00, 00, time.UTC),
		time.Date(2019, 02, 12, 00, 00, 00, 00, time.UTC),
		time.Date(2019, 02, 13, 00, 00, 00, 00, time.UTC),
	}

	sort.Sort(Descending(times))
	assert.Equal(13, times[0].Day())
}
