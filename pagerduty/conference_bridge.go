/*

Copyright (c) 2022 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.

*/

package pagerduty

// ConferenceBridge is an api type.
type ConferenceBridge struct {
	ConferenceNumber string `json:"conference_number,omitempty"`
	ConferenceURL    string `json:"conference_url,omitempty"`
}
