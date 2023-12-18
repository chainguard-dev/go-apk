// Copyright 2023 Chainguard, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apk

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			version  string
			expected packageVersion
		}{
			// various legitimate ones
			{"1", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1.1", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1a", packageVersion{numbers: []int{1}, letter: 'a', preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1a", packageVersion{numbers: []int{1, 1}, letter: 'a', preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1.1a", packageVersion{numbers: []int{1, 1, 1}, letter: 'a', preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1_alpha", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1_beta", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierBeta, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1_alpha1", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1_alpha2", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1_alpha", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1.1_alpha", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1_alpha1", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1a_alpha1", packageVersion{numbers: []int{1}, letter: 'a', preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1a_alpha2", packageVersion{numbers: []int{1}, letter: 'a', preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1b_alpha", packageVersion{numbers: []int{1, 1}, letter: 'b', preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1.1c_alpha", packageVersion{numbers: []int{1, 1, 1}, letter: 'c', preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1r_alpha1", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, letter: 'r', postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1.1.1s_alpha2", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, letter: 's', postSuffix: packageVersionPostModifierNone, revision: 0}},
			{"1-r2", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1-r2", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1-r2", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1a-r2", packageVersion{numbers: []int{1}, letter: 'a', preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1a-r2", packageVersion{numbers: []int{1, 1}, letter: 'a', preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1a-r2", packageVersion{numbers: []int{1, 1, 1}, letter: 'a', preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1_alpha-r2", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1_beta-r2", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierBeta, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1_alpha1-r2", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1_alpha2-r2", packageVersion{numbers: []int{1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1_alpha-r2", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1_alpha-r2", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1_alpha1-r2", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1_alpha2-r2", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1a_alpha1-r2", packageVersion{numbers: []int{1}, letter: 'a', preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1a_alpha2-r2", packageVersion{numbers: []int{1}, letter: 'a', preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1b_alpha-r2", packageVersion{numbers: []int{1, 1}, letter: 'b', preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1c_alpha-r2", packageVersion{numbers: []int{1, 1, 1}, letter: 'c', preSuffix: packageVersionPreModifierAlpha, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1r_alpha1-r2", packageVersion{numbers: []int{1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 1, letter: 'r', postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1s_alpha2-r2", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierAlpha, preSuffixNumber: 2, letter: 's', postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1-r2", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 2}},
			{"1.1.1-r29", packageVersion{numbers: []int{1, 1, 1}, preSuffix: packageVersionPreModifierNone, postSuffix: packageVersionPostModifierNone, revision: 29}},
		}
		for _, tt := range tests {
			actual, err := parseVersion(tt.version)
			require.NoError(t, err, "%q unexpected error", tt.version)
			require.Equal(t, tt.expected, actual, "%q expected %v, got %v", tt.version, tt.expected, actual)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		tests := []string{
			// various illegitimate ones
			"a.1.2",
			"1.a.2",
			"1_illegal",
			"1_illegal",
			"1.1.1-rQ",
		}
		for _, version := range tests {
			_, err := parseVersion(version)
			require.Error(t, err, "%q mismatched error", version)
		}
	})
}

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		versionA string
		expected versionCompare
		versionB string
	}{
		{"2.34", greater, "0.1.0_alpha"},
		{"0.1.0_alpha", equal, "0.1.0_alpha"},
		{"0.1.0_alpha", less, "0.1.3_alpha"},
		{"0.1.3_alpha", greater, "0.1.0_alpha"},
		{"0.1.0_alpha2", greater, "0.1.0_alpha"},
		{"0.1.0_alpha", less, "2.2.39-r1"},
		{"2.2.39-r1", greater, "1.0.4-r3"},
		{"1.0.4-r3", less, "1.0.4-r4"},
		{"1.0.4-r4", less, "1.6"},
		{"1.6", greater, "1.0.2"},
		{"1.0.2", greater, "0.7-r1"},
		{"0.7-r1", less, "1.0.0"},
		{"1.0.0", less, "1.0.1"},
		{"1.0.1", less, "1.1"},
		{"1.1", greater, "1.1_alpha1"},
		{"1.1_alpha1", less, "1.2.1"},
		{"1.2.1", greater, "1.2"},
		{"1.2", less, "1.3_alpha"},
		{"1.3_alpha", less, "1.3_alpha2"},
		{"1.3_alpha2", less, "1.3_alpha3"},
		{"1.3_alpha8", greater, "0.6.0"},
		{"0.6.0", less, "0.6.1"},
		{"0.6.1", less, "0.7.0"},
		{"0.7.0", less, "0.8_beta1"},
		{"0.8_beta1", less, "0.8_beta2"},
		{"0.8_beta4", less, "4.8-r1"},
		{"4.8-r1", greater, "3.10.18-r1"},
		{"3.10.18-r1", greater, "2.3.0b-r1"},
		{"2.3.0b-r1", less, "2.3.0b-r2"},
		{"2.3.0b-r2", less, "2.3.0b-r3"},
		{"2.3.0b-r3", less, "2.3.0b-r4"},
		{"2.3.0b-r4", greater, "0.12.1"},
		{"0.12.1", less, "0.12.2"},
		{"0.12.2", less, "0.12.3"},
		{"0.12.3", greater, "0.12"},
		{"0.12", less, "0.13_beta1"},
		{"0.13_beta1", less, "0.13_beta2"},
		{"0.13_beta2", less, "0.13_beta3"},
		{"0.13_beta3", less, "0.13_beta4"},
		{"0.13_beta4", less, "0.13_beta5"},
		{"0.13_beta5", greater, "0.9.12"},
		{"0.9.12", less, "0.9.13"},
		{"0.9.13", greater, "0.9.12"},
		{"0.9.12", less, "0.9.13"},
		{"0.9.13", greater, "0.0.16"},
		{"0.0.16", less, "0.6"},
		{"0.6", less, "2.1.13-r3"},
		{"2.1.13-r3", less, "2.1.15-r2"},
		{"2.1.15-r2", less, "2.1.15-r3"},
		{"2.1.15-r3", greater, "1.2.11"},
		{"1.2.11", less, "1.2.12.1"},
		{"1.2.12.1", less, "1.2.13"},
		{"1.2.13", less, "1.2.14-r1"},
		{"1.2.14-r1", greater, "0.7.1"},
		{"0.7.1", greater, "0.5.4"},
		{"0.5.4", less, "0.7.0"},
		{"0.7.0", less, "1.2.13"},
		{"1.2.13", greater, "1.0.8"},
		{"1.0.8", less, "1.2.1"},
		{"1.2.1", greater, "0.7-r1"},
		{"0.7-r1", less, "2.4.32"},
		{"2.4.32", less, "2.8-r4"},
		{"2.8-r4", greater, "0.9.6"},
		{"0.9.6", greater, "0.2.0-r1"},
		{"0.2.0-r1", equal, "0.2.0-r1"},
		{"0.2.0-r1", less, "3.1_p16"},
		{"3.1_p16", less, "3.1_p17"},
		{"3.1_p17", greater, "1.06-r6"},
		{"1.06-r6", less, "006"},
		{"006", greater, "1.0.0"},
		{"1.0.0", less, "1.2.2-r1"},
		{"1.2.2-r1", greater, "1.2.2"},
		{"1.2.2", greater, "0.3-r1"},
		{"0.3-r1", less, "9.3.2-r4"},
		{"9.3.2-r4", less, "9.3.4-r2"},
		{"9.3.4-r2", greater, "9.3.4"},
		{"9.3.4", greater, "9.3.2"},
		{"9.3.2", less, "9.3.4"},
		{"9.3.4", greater, "1.1.3"},
		{"1.1.3", less, "2.16.1-r3"},
		{"2.16.1-r3", equal, "2.16.1-r3"},
		{"2.16.1-r3", greater, "2.1.0-r2"},
		{"2.1.0-r2", less, "2.9.3-r1"},
		{"2.9.3-r1", greater, "0.9-r1"},
		{"0.9-r1", greater, "0.8-r1"},
		{"0.8-r1", less, "1.0.6-r3"},
		{"1.0.6-r3", greater, "0.11"},
		{"0.11", less, "0.12"},
		{"0.12", less, "1.2.1-r1"},
		{"1.2.1-r1", less, "1.2.2.1"},
		{"1.2.2.1", less, "1.4.1-r1"},
		{"1.4.1-r1", less, "1.4.1-r2"},
		{"1.4.1-r2", greater, "1.2.2"},
		{"1.2.2", less, "1.3"},
		{"1.3", greater, "1.0.3-r6"},
		{"1.0.3-r6", less, "1.0.4"},
		{"1.0.4", less, "2.59"},
		{"2.59", less, "20050718-r1"},
		{"20050718-r1", less, "20050718-r2"},
		{"20050718-r2", greater, "3.9.8-r5"},
		{"3.9.8-r5", greater, "2.01.01_alpha10"},
		{"2.01.01_alpha10", greater, "0.94"},
		{"0.94", less, "1.0"},
		{"1.0", greater, "0.99.3.20040818"},
		{"0.99.3.20040818", greater, "0.7"},
		{"0.7", less, "1.21-r1"},
		{"1.21-r1", greater, "0.13"},
		{"0.13", less, "0.90.1-r1"},
		{"0.90.1-r1", greater, "0.10.2"},
		{"0.10.2", less, "0.10.3"},
		{"0.10.3", less, "1.6"},
		{"1.6", less, "1.39"},
		{"1.39", greater, "1.00_beta2"},
		{"1.00_beta2", greater, "0.9.2"},
		{"0.9.2", less, "5.94-r1"},
		{"5.94-r1", less, "6.4"},
		{"6.4", greater, "2.6-r5"},
		{"2.6-r5", greater, "1.4"},
		{"1.4", less, "2.8.9-r1"},
		{"2.8.9-r1", greater, "2.8.9"},
		{"2.8.9", greater, "1.1"},
		{"1.1", greater, "1.0.3-r2"},
		{"1.0.3-r2", less, "1.3.4-r3"},
		{"1.3.4-r3", less, "2.2"},
		{"2.2", greater, "1.2.6"},
		{"1.2.6", less, "7.15.1-r1"},
		{"7.15.1-r1", greater, "1.02"},
		{"1.02", less, "1.03-r1"},
		{"1.03-r1", less, "1.12.12-r2"},
		{"1.12.12-r2", less, "2.8.0.6-r1"},
		{"2.8.0.6-r1", greater, "0.5.2.7"},
		{"0.5.2.7", less, "4.2.52_p2-r1"},
		{"4.2.52_p2-r1", less, "4.2.52_p4-r2"},
		{"4.2.52_p4-r2", greater, "1.02.07"},
		{"1.02.07", less, "1.02.10-r1"},
		{"1.02.10-r1", less, "3.0.3-r9"},
		{"3.0.3-r9", greater, "2.0.5-r1"},
		{"2.0.5-r1", less, "4.5"},
		{"4.5", greater, "2.8.7-r1"},
		{"2.8.7-r1", greater, "1.0.5"},
		{"1.0.5", less, "8"},
		{"8", less, "9"},
		{"9", greater, "2.18.3-r10"},
		{"2.18.3-r10", greater, "1.05-r18"},
		{"1.05-r18", less, "1.05-r19"},
		{"1.05-r19", less, "2.2.5"},
		{"2.2.5", less, "2.8"},
		{"2.8", less, "2.20.1"},
		{"2.20.1", less, "2.20.3"},
		{"2.20.3", less, "2.31"},
		{"2.31", less, "2.34"},
		{"2.34", less, "2.38"},
		{"2.38", less, "20050405"},
		{"20050405", greater, "1.8"},
		{"1.8", less, "2.11-r1"},
		{"2.11-r1", greater, "2.11"},
		{"2.11", greater, "0.1.6-r3"},
		{"0.1.6-r3", less, "0.47-r1"},
		{"0.47-r1", less, "0.49"},
		{"0.49", less, "3.6.8-r2"},
		{"3.6.8-r2", greater, "1.39"},
		{"1.39", less, "2.43"},
		{"2.43", greater, "2.0.6-r1"},
		{"2.0.6-r1", greater, "0.2-r6"},
		{"0.2-r6", less, "0.4"},
		{"0.4", less, "1.0.0"},
		{"1.0.0", less, "10-r1"},
		{"10-r1", greater, "4"},
		{"4", greater, "0.7.3-r2"},
		{"0.7.3-r2", greater, "0.7.3"},
		{"0.7.3", less, "1.95.8"},
		{"1.95.8", greater, "1.1.19"},
		{"1.1.19", greater, "1.1.5"},
		{"1.1.5", less, "6.3.2-r1"},
		{"6.3.2-r1", less, "6.3.3"},
		{"6.3.3", greater, "4.17-r1"},
		{"4.17-r1", less, "4.18"},
		{"4.18", less, "4.19"},
		{"4.19", greater, "4.3.0"},
		{"4.3.0", less, "4.3.2-r1"},
		{"4.3.2-r1", greater, "4.3.2"},
		{"4.3.2", greater, "0.68-r3"},
		{"0.68-r3", less, "1.0.0"},
		{"1.0.0", less, "1.0.1"},
		{"1.0.1", greater, "1.0.0"},
		{"1.0.0", equal, "1.0.0"},
		{"1.0.0", less, "1.0.1"},
		{"1.0.1", less, "2.3.2-r1"},
		{"2.3.2-r1", less, "2.4.2"},
		{"2.4.2", less, "20060720"},
		{"20060720", greater, "3.0.20060720"},
		{"3.0.20060720", less, "20060720"},
		{"20060720", greater, "1.1"},
		{"1.1", equal, "1.1"},
		{"1.1", less, "1.1.1-r1"},
		{"1.1.1-r1", less, "1.1.3-r1"},
		{"1.1.3-r1", less, "1.1.3-r2"},
		{"1.1.3-r2", less, "2.1.10-r2"},
		{"2.1.10-r2", greater, "0.7.18-r2"},
		{"0.7.18-r2", less, "0.17-r6"},
		{"0.17-r6", less, "2.6.1"},
		{"2.6.1", less, "2.6.3"},
		{"2.6.3", less, "3.1.5-r2"},
		{"3.1.5-r2", less, "3.4.6-r1"},
		{"3.4.6-r1", less, "3.4.6-r2"},
		{"3.4.6-r2", equal, "3.4.6-r2"},
		{"3.4.6-r2", greater, "2.0.33"},
		{"2.0.33", less, "2.0.34"},
		{"2.0.34", greater, "1.8.3-r2"},
		{"1.8.3-r2", less, "1.8.3-r3"},
		{"1.8.3-r3", less, "4.1"},
		{"4.1", less, "8.54"},
		{"8.54", greater, "4.1.4"},
		{"4.1.4", greater, "1.2.10-r5"},
		{"1.2.10-r5", less, "4.1.4-r3"},
		{"4.1.4-r3", equal, "4.1.4-r3"},
		{"4.1.4-r3", less, "4.2.1"},
		{"4.2.1", greater, "4.1.0"},
		{"4.1.0", less, "8.11"},
		{"8.11", greater, "1.4.4-r1"},
		{"1.4.4-r1", less, "2.1.9.200602141850"},
		{"2.1.9.200602141850", greater, "1.6"},
		{"1.6", less, "2.5.1-r8"},
		{"2.5.1-r8", less, "2.5.1a-r1"},
		{"2.5.1a-r1", greater, "1.19.2-r1"},
		{"1.19.2-r1", greater, "0.97-r2"},
		{"0.97-r2", less, "0.97-r3"},
		{"0.97-r3", less, "1.3.5-r10"},
		{"1.3.5-r10", greater, "1.3.5-r8"},
		{"1.3.5-r8", less, "1.3.5-r9"},
		{"1.3.5-r9", greater, "1.0"},
		{"1.0", less, "1.1"},
		{"1.1", greater, "0.9.11"},
		{"0.9.11", less, "0.9.12"},
		{"0.9.12", less, "0.9.13"},
		{"0.9.13", less, "0.9.14"},
		{"0.9.14", less, "0.9.15"},
		{"0.9.15", less, "0.9.16"},
		{"0.9.16", greater, "0.3-r2"},
		{"0.3-r2", less, "6.3"},
		{"6.3", less, "6.6"},
		{"6.6", less, "6.9"},
		{"6.9", greater, "0.7.2-r3"},
		{"0.7.2-r3", less, "1.2.10"},
		{"1.2.10", less, "20040923-r2"},
		{"20040923-r2", greater, "20040401"},
		{"20040401", greater, "2.0.0_rc3-r1"},
		{"2.0.0_rc3-r1", greater, "1.5"},
		{"1.5", less, "4.4"},
		{"4.4", greater, "1.0.1"},
		{"1.0.1", less, "2.2.0"},
		{"2.2.0", greater, "1.1.0-r2"},
		{"1.1.0-r2", greater, "0.3"},
		{"0.3", less, "20020207-r2"},
		{"20020207-r2", greater, "1.31-r2"},
		{"1.31-r2", less, "3.7"},
		{"3.7", greater, "2.0.1"},
		{"2.0.1", less, "2.0.2"},
		{"2.0.2", greater, "0.99.163"},
		{"0.99.163", less, "2.6.15.20060110"},
		{"2.6.15.20060110", less, "2.6.16.20060323"},
		{"2.6.16.20060323", less, "2.6.19.20061214"},
		{"2.6.19.20061214", greater, "0.6.2-r1"},
		{"0.6.2-r1", less, "0.6.3"},
		{"0.6.3", less, "0.6.5"},
		{"0.6.5", less, "1.3.5-r1"},
		{"1.3.5-r1", less, "1.3.5-r4"},
		{"1.3.5-r4", less, "3.0.0-r2"},
		{"3.0.0-r2", less, "021109-r3"},
		{"021109-r3", less, "20060512"},
		{"20060512", greater, "1.24"},
		{"1.24", greater, "0.9.16-r1"},
		{"0.9.16-r1", less, "3.9_pre20060124"},
		{"3.9_pre20060124", greater, "0.01"},
		{"0.01", less, "0.06"},
		{"0.06", less, "1.1.7"},
		{"1.1.7", less, "6b-r7"},
		{"6b-r7", greater, "1.12-r7"},
		{"1.12-r7", less, "1.12-r8"},
		{"1.12-r8", greater, "1.1.12"},
		{"1.1.12", less, "1.1.13"},
		{"1.1.13", greater, "0.3"},
		{"0.3", less, "0.5"},
		{"0.5", less, "3.96.1"},
		{"3.96.1", less, "3.97"},
		{"3.97", greater, "0.10.0-r1"},
		{"0.10.0-r1", greater, "0.10.0"},
		{"0.10.0", less, "0.10.1_rc1"},
		{"0.10.1_rc1", greater, "0.9.11"},
		{"0.9.11", less, "394"},
		{"394", greater, "2.31"},
		{"2.31", greater, "1.0.1"},
		{"1.0.1", equal, "1.0.1"},
		{"1.0.1", less, "1.0.3"},
		{"1.0.3", greater, "1.0.2"},
		{"1.0.2", equal, "1.0.2"},
		{"1.0.2", greater, "1.0.1"},
		{"1.0.1", equal, "1.0.1"},
		{"1.0.1", less, "1.2.2"},
		{"1.2.2", less, "2.1.10"},
		{"2.1.10", greater, "1.0.1"},
		{"1.0.1", less, "1.0.2"},
		{"1.0.2", less, "3.5.5"},
		{"3.5.5", greater, "1.1.1"},
		{"1.1.1", greater, "0.9.1"},
		{"0.9.1", less, "1.0.2"},
		{"1.0.2", greater, "1.0.1"},
		{"1.0.1", less, "1.0.2"},
		{"1.0.2", greater, "1.0.1"},
		{"1.0.1", equal, "1.0.1"},
		{"1.0.1", less, "1.0.5"},
		{"1.0.5", greater, "0.8.5"},
		{"0.8.5", less, "0.8.6-r3"},
		{"0.8.6-r3", less, "2.3.17"},
		{"2.3.17", greater, "1.10-r5"},
		{"1.10-r5", less, "1.10-r9"},
		{"1.10-r9", less, "2.0.2"},
		{"2.0.2", greater, "1.1a"},
		{"1.1a", less, "1.3a"},
		{"1.3a", greater, "1.0.2"},
		{"1.0.2", less, "1.2.2-r1"},
		{"1.2.2-r1", greater, "1.0-r1"},
		{"1.0-r1", greater, "0.15.1b"},
		{"0.15.1b", less, "1.0.1"},
		{"1.0.1", less, "1.06-r1"},
		{"1.06-r1", less, "1.06-r2"},
		{"1.06-r2", greater, "0.15.1b-r2"},
		{"0.15.1b-r2", greater, "0.15.1b"},
		{"0.15.1b", less, "2.5.7"},
		{"2.5.7", greater, "1.1.2.1-r1"},
		{"1.1.2.1-r1", greater, "0.0.31"},
		{"0.0.31", less, "0.0.50"},
		{"0.0.50", greater, "0.0.16"},
		{"0.0.16", less, "0.0.25"},
		{"0.0.25", less, "0.17"},
		{"0.17", greater, "0.5.0"},
		{"0.5.0", less, "1.1.2"},
		{"1.1.2", less, "1.1.3"},
		{"1.1.3", less, "1.1.20"},
		{"1.1.20", greater, "0.9.4"},
		{"0.9.4", less, "0.9.5"},
		{"0.9.5", less, "6.3"},
		{"6.3", less, "6.6"},
		{"6.6", greater, "6.3"},
		{"6.3", less, "6.6"},
		{"6.6", greater, "1.2.12-r1"},
		{"1.2.12-r1", less, "1.2.13"},
		{"1.2.13", less, "1.2.14"},
		{"1.2.14", less, "1.2.15"},
		{"1.2.15", less, "8.0.12"},
		{"8.0.12", greater, "8.0.9"},
		{"8.0.9", greater, "1.2.3-r1"},
		{"1.2.3-r1", less, "1.2.4-r1"},
		{"1.2.4-r1", greater, "0.1"},
		{"0.1", less, "0.3.5"},
		{"0.3.5", less, "1.5.22"},
		{"1.5.22", greater, "0.1.11"},
		{"0.1.11", less, "0.1.12"},
		{"0.1.12", less, "1.1.4.1"},
		{"1.1.4.1", greater, "1.1.0"},
		{"1.1.0", less, "1.1.2"},
		{"1.1.2", greater, "1.0.3"},
		{"1.0.3", greater, "1.0.2"},
		{"1.0.2", less, "2.6.26"},
		{"2.6.26", less, "2.6.27"},
		{"2.6.27", greater, "1.1.17"},
		{"1.1.17", less, "1.4.11"},
		{"1.4.11", less, "22.7-r1"},
		{"22.7-r1", less, "22.7.3-r1"},
		{"22.7.3-r1", greater, "22.7"},
		{"22.7", greater, "2.1_pre20"},
		{"2.1_pre20", less, "2.1_pre26"},
		{"2.1_pre26", greater, "0.2.3-r2"},
		{"0.2.3-r2", greater, "0.2.2"},
		{"0.2.2", less, "2.10.0"},
		{"2.10.0", less, "2.10.1"},
		{"2.10.1", greater, "02.08.01b"},
		{"02.08.01b", less, "4.77"},
		{"4.77", greater, "0.17"},
		{"0.17", less, "5.1.1-r1"},
		{"5.1.1-r1", less, "5.1.1-r2"},
		{"5.1.1-r2", greater, "5.1.1"},
		{"5.1.1", greater, "1.2"},
		{"1.2", less, "5.1"},
		{"5.1", greater, "2.02.06"},
		{"2.02.06", less, "2.02.10"},
		{"2.02.10", less, "2.8.5-r3"},
		{"2.8.5-r3", less, "2.8.6-r1"},
		{"2.8.6-r1", less, "2.8.6-r2"},
		{"2.8.6-r2", greater, "2.02-r1"},
		{"2.02-r1", greater, "1.5.0-r1"},
		{"1.5.0-r1", greater, "1.5.0"},
		{"1.5.0", greater, "0.9.2"},
		{"0.9.2", less, "8.1.2.20040524-r1"},
		{"8.1.2.20040524-r1", less, "8.1.2.20050715-r1"},
		{"8.1.2.20050715-r1", less, "20030215"},
		{"20030215", greater, "3.80-r4"},
		{"3.80-r4", less, "3.81"},
		{"3.81", greater, "1.6d"},
		{"1.6d", greater, "1.2.07.8"},
		{"1.2.07.8", less, "1.2.12.04"},
		{"1.2.12.04", less, "1.2.12.05"},
		{"1.2.12.05", less, "1.3.3"},
		{"1.3.3", less, "2.6.4"},
		{"2.6.4", greater, "2.5.2"},
		{"2.5.2", less, "2.6.1"},
		{"2.6.1", greater, "2.6"},
		{"2.6", less, "6.5.1-r1"},
		{"6.5.1-r1", greater, "1.1.35-r1"},
		{"1.1.35-r1", less, "1.1.35-r2"},
		{"1.1.35-r2", greater, "0.9.2"},
		{"0.9.2", less, "1.07-r1"},
		{"1.07-r1", less, "1.07.5"},
		{"1.07.5", greater, "1.07"},
		{"1.07", less, "1.19"},
		{"1.19", less, "2.1-r2"},
		{"2.1-r2", less, "2.2"},
		{"2.2", greater, "1.0.4"},
		{"1.0.4", less, "20060811"},
		{"20060811", less, "20061003"},
		{"20061003", greater, "0.1_pre20060810"},
		{"0.1_pre20060810", less, "0.1_pre20060817"},
		{"0.1_pre20060817", less, "1.0.3"},
		{"1.0.3", greater, "1.0.2"},
		{"1.0.2", greater, "1.0.1"},
		{"1.0.1", less, "3.2.2-r1"},
		{"3.2.2-r1", less, "3.2.2-r2"},
		{"3.2.2-r2", less, "3.3.17"},
		{"3.3.17", greater, "0.59s-r11"},
		{"0.59s-r11", less, "0.65"},
		{"0.65", greater, "0.2.10-r2"},
		{"0.2.10-r2", less, "2.01"},
		{"2.01", less, "3.9.10"},
		{"3.9.10", greater, "1.2.18"},
		{"1.2.18", less, "1.5.11-r2"},
		{"1.5.11-r2", less, "1.5.13-r1"},
		{"1.5.13-r1", greater, "1.3.12-r1"},
		{"1.3.12-r1", less, "2.0.1"},
		{"2.0.1", less, "2.0.2"},
		{"2.0.2", less, "2.0.3"},
		{"2.0.3", greater, "0.2.0"},
		{"0.2.0", less, "5.5-r2"},
		{"5.5-r2", less, "5.5-r3"},
		{"5.5-r3", greater, "0.25.3"},
		{"0.25.3", less, "0.26.1-r1"},
		{"0.26.1-r1", less, "5.2.1.2-r1"},
		{"5.2.1.2-r1", less, "5.4"},
		{"5.4", greater, "1.60-r11"},
		{"1.60-r11", less, "1.60-r12"},
		{"1.60-r12", less, "110-r8"},
		{"110-r8", greater, "0.17-r2"},
		{"0.17-r2", less, "1.05-r4"},
		{"1.05-r4", less, "5.28.0"},
		{"5.28.0", greater, "0.51.6-r1"},
		{"0.51.6-r1", less, "1.0.6-r6"},
		{"1.0.6-r6", greater, "0.8.3"},
		{"0.8.3", less, "1.42"},
		{"1.42", less, "20030719"},
		{"20030719", greater, "4.01"},
		{"4.01", less, "4.20"},
		{"4.20", greater, "0.20070118"},
		{"0.20070118", less, "0.20070207_rc1"},
		{"0.20070207_rc1", less, "1.0"},
		{"1.0", less, "1.13.0"},
		{"1.13.0", less, "1.13.1"},
		{"1.13.1", greater, "0.21"},
		{"0.21", greater, "0.3.7-r3"},
		{"0.3.7-r3", less, "0.4.10"},
		{"0.4.10", less, "0.5.0"},
		{"0.5.0", less, "0.5.5"},
		{"0.5.5", less, "0.5.7"},
		{"0.5.7", less, "0.6.11-r1"},
		{"0.6.11-r1", less, "2.3.30-r2"},
		{"2.3.30-r2", less, "3.7_p1"},
		{"3.7_p1", greater, "1.3"},
		{"1.3", greater, "0.10.1"},
		{"0.10.1", less, "4.3_p2-r1"},
		{"4.3_p2-r1", less, "4.3_p2-r5"},
		{"4.3_p2-r5", less, "4.4_p1-r6"},
		{"4.4_p1-r6", less, "4.5_p1-r1"},
		{"4.5_p1-r1", greater, "4.5_p1"},
		{"4.5_p1", less, "4.5_p1-r1"},
		{"4.5_p1-r1", greater, "4.5_p1"},
		{"4.5_p1", greater, "0.9.8c-r1"},
		{"0.9.8c-r1", less, "0.9.8d"},
		{"0.9.8d", less, "2.4.4"},
		{"2.4.4", less, "2.4.7"},
		{"2.4.7", greater, "2.0.6"},
		{"2.0.6", equal, "2.0.6"},
		{"2.0.6", greater, "0.78-r3"},
		{"0.78-r3", greater, "0.3.2"},
		{"0.3.2", less, "1.7.1-r1"},
		{"1.7.1-r1", less, "2.5.9"},
		{"2.5.9", greater, "0.1.13"},
		{"0.1.13", less, "0.1.15"},
		{"0.1.15", less, "0.4"},
		{"0.4", less, "0.9.6"},
		{"0.9.6", less, "2.2.0-r1"},
		{"2.2.0-r1", less, "2.2.3-r2"},
		{"2.2.3-r2", less, "013"},
		{"013", less, "014-r1"},
		{"014-r1", greater, "1.3.1-r1"},
		{"1.3.1-r1", less, "5.8.8-r2"},
		{"5.8.8-r2", greater, "5.1.6-r4"},
		{"5.1.6-r4", less, "5.1.6-r6"},
		{"5.1.6-r6", less, "5.2.1-r3"},
		{"5.2.1-r3", greater, "0.11.3"},
		{"0.11.3", equal, "0.11.3"},
		{"0.11.3", less, "1.10.7"},
		{"1.10.7", greater, "1.7-r1"},
		{"1.7-r1", greater, "0.1.20"},
		{"0.1.20", less, "0.1.23"},
		{"0.1.23", less, "5b-r9"},
		{"5b-r9", greater, "2.2.10"},
		{"2.2.10", less, "2.3.6"},
		{"2.3.6", less, "8.0.12"},
		{"8.0.12", greater, "2.4.3-r16"},
		{"2.4.3-r16", less, "2.4.4-r4"},
		{"2.4.4-r4", less, "3.0.3-r5"},
		{"3.0.3-r5", less, "3.0.6"},
		{"3.0.6", less, "3.2.6"},
		{"3.2.6", less, "3.2.7"},
		{"3.2.7", greater, "0.3.1_rc8"},
		{"0.3.1_rc8", less, "22.2"},
		{"22.2", less, "22.3"},
		{"22.3", greater, "1.2.2"},
		{"1.2.2", less, "2.04"},
		{"2.04", less, "2.4.3-r1"},
		{"2.4.3-r1", less, "2.4.3-r4"},
		{"2.4.3-r4", greater, "0.98.6-r1"},
		{"0.98.6-r1", less, "5.7-r2"},
		{"5.7-r2", less, "5.7-r3"},
		{"5.7-r3", greater, "5.1_p4"},
		{"5.1_p4", greater, "1.0.5"},
		{"1.0.5", less, "3.6.19-r1"},
		{"3.6.19-r1", greater, "3.6.19"},
		{"3.6.19", greater, "1.0.1"},
		{"1.0.1", less, "3.8"},
		{"3.8", greater, "0.2.3"},
		{"0.2.3", less, "1.2.15-r3"},
		{"1.2.15-r3", greater, "1.2.6-r1"},
		{"1.2.6-r1", less, "2.6.8-r2"},
		{"2.6.8-r2", less, "2.6.9-r1"},
		{"2.6.9-r1", greater, "1.7"},
		{"1.7", less, "1.7b"},
		{"1.7b", less, "1.8.4-r3"},
		{"1.8.4-r3", less, "1.8.5"},
		// FIXME(kaniini): _p2 is different than _pre2.
		// {"1.8.5", less, "1.8.5_p2"},
		{"1.8.5_p2", greater, "1.1.3"},
		{"1.1.3", less, "3.0.22-r3"},
		{"3.0.22-r3", less, "3.0.24"},
		{"3.0.24", equal, "3.0.24"},
		{"3.0.24", equal, "3.0.24"},
		{"3.0.24", less, "4.0.2-r5"},
		{"4.0.2-r5", less, "4.0.3"},
		{"4.0.3", greater, "0.98"},
		{"0.98", less, "1.00"},
		{"1.00", less, "4.1.4-r1"},
		{"4.1.4-r1", less, "4.1.5"},
		{"4.1.5", greater, "2.3"},
		{"2.3", less, "2.17-r3"},
		{"2.17-r3", greater, "0.1.7"},
		{"0.1.7", less, "1.11"},
		{"1.11", less, "4.2.1-r11"},
		{"4.2.1-r11", greater, "3.2.3"},
		{"3.2.3", less, "3.2.4"},
		{"3.2.4", less, "3.2.8"},
		{"3.2.8", less, "3.2.9"},
		{"3.2.9", greater, "3.2.3"},
		{"3.2.3", less, "3.2.4"},
		{"3.2.4", less, "3.2.8"},
		{"3.2.8", less, "3.2.9"},
		{"3.2.9", greater, "1.4.9-r2"},
		{"1.4.9-r2", less, "2.9.11_pre20051101-r2"},
		{"2.9.11_pre20051101-r2", less, "2.9.11_pre20051101-r3"},
		{"2.9.11_pre20051101-r3", greater, "2.9.11_pre20051101"},
		{"2.9.11_pre20051101", less, "2.9.11_pre20061021-r1"},
		{"2.9.11_pre20061021-r1", less, "2.9.11_pre20061021-r2"},
		{"2.9.11_pre20061021-r2", less, "5.36-r1"},
		{"5.36-r1", greater, "1.0.1"},
		{"1.0.1", less, "7.0-r2"},
		{"7.0-r2", greater, "2.4.5"},
		{"2.4.5", less, "2.6.1.2"},
		{"2.6.1.2", less, "2.6.1.3-r1"},
		{"2.6.1.3-r1", greater, "2.6.1.3"},
		{"2.6.1.3", less, "2.6.1.3-r1"},
		{"2.6.1.3-r1", less, "12.17.9"},
		{"12.17.9", greater, "1.1.12"},
		{"1.1.12", greater, "1.1.7"},
		{"1.1.7", less, "2.5.14"},
		{"2.5.14", less, "2.6.6-r1"},
		{"2.6.6-r1", less, "2.6.7"},
		{"2.6.7", less, "2.6.9-r1"},
		{"2.6.9-r1", greater, "2.6.9"},
		{"2.6.9", greater, "1.39"},
		{"1.39", greater, "0.9"},
		{"0.9", less, "2.61-r2"},
		{"2.61-r2", less, "4.5.14"},
		// TODO(kaniini): Fix 4.5.14 > 4.09
		// {"4.5.14", greater, "4.09-r1"},
		{"4.09-r1", greater, "1.3.1"},
		{"1.3.1", less, "1.3.2-r3"},
		{"1.3.2-r3", less, "1.6.8_p12-r1"},
		{"1.6.8_p12-r1", greater, "1.6.8_p9-r2"},
		{"1.6.8_p9-r2", greater, "1.3.0-r1"},
		{"1.3.0-r1", less, "3.11"},
		{"3.11", less, "3.20"},
		{"3.20", greater, "1.6.11-r1"},
		{"1.6.11-r1", greater, "1.6.9"},
		{"1.6.9", less, "5.0.5-r2"},
		{"5.0.5-r2", greater, "2.86-r5"},
		{"2.86-r5", less, "2.86-r6"},
		{"2.86-r6", greater, "1.15.1-r1"},
		{"1.15.1-r1", less, "8.4.9"},
		{"8.4.9", greater, "7.6-r8"},
		{"7.6-r8", greater, "3.9.4-r2"},
		{"3.9.4-r2", less, "3.9.4-r3"},
		{"3.9.4-r3", less, "3.9.5-r2"},
		{"3.9.5-r2", greater, "1.1.9"},
		{"1.1.9", greater, "1.0.6"},
		{"1.0.6", less, "5.9"},
		{"5.9", less, "6.5"},
		{"6.5", greater, "0.40-r1"},
		{"0.40-r1", less, "2.25b-r5"},
		{"2.25b-r5", less, "2.25b-r6"},
		{"2.25b-r6", greater, "1.0.4"},
		{"1.0.4", less, "1.0.5"},
		{"1.0.5", less, "1.4_p12-r2"},
		{"1.4_p12-r2", less, "1.4_p12-r5"},
		{"1.4_p12-r5", greater, "1.1"},
		{"1.1", greater, "0.2.0-r1"},
		{"0.2.0-r1", less, "0.2.1"},
		{"0.2.1", less, "0.9.28-r1"},
		{"0.9.28-r1", less, "0.9.28-r2"},
		{"0.9.28-r2", less, "0.9.28.1"},
		{"0.9.28.1", greater, "0.9.28"},
		{"0.9.28", less, "0.9.28.1"},
		{"0.9.28.1", less, "087-r1"},
		{"087-r1", less, "103"},
		{"103", less, "104-r11"},
		{"104-r11", greater, "104-r9"},
		{"104-r9", greater, "1.23-r1"},
		{"1.23-r1", greater, "1.23"},
		{"1.23", less, "1.23-r1"},
		{"1.23-r1", greater, "1.0.2"},
		{"1.0.2", less, "5.52-r1"},
		{"5.52-r1", greater, "1.2.5_rc2"},
		{"1.2.5_rc2", greater, "0.1"},
		{"0.1", less, "0.71-r1"},
		{"0.71-r1", less, "20040406-r1"},
		{"20040406-r1", greater, "2.12r-r4"},
		{"2.12r-r4", less, "2.12r-r5"},
		{"2.12r-r5", greater, "0.0.7"},
		{"0.0.7", less, "1.0.3"},
		{"1.0.3", less, "1.8"},
		{"1.8", less, "7.0.17"},
		{"7.0.17", less, "7.0.174"},
		{"7.0.174", greater, "7.0.17"},
		{"7.0.17", less, "7.0.174"},
		{"7.0.174", greater, "1.0.1"},
		{"1.0.1", less, "1.1.1-r3"},
		{"1.1.1-r3", greater, "0.3.4_pre20061029"},
		{"0.3.4_pre20061029", less, "0.4.0"},
		{"0.4.0", greater, "0.1.2"},
		{"0.1.2", less, "1.10.2"},
		{"1.10.2", less, "2.16"},
		{"2.16", less, "28"},
		{"28", greater, "0.99.4"},
		{"0.99.4", less, "1.13"},
		{"1.13", greater, "1.0.1"},
		{"1.0.1", less, "1.1.2-r2"},
		{"1.1.2-r2", greater, "1.1.0"},
		{"1.1.0", less, "1.1.1"},
		{"1.1.1", equal, "1.1.1"},
		{"1.1.1", greater, "0.6.0"},
		{"0.6.0", less, "6.6.3"},
		{"6.6.3", greater, "1.1.1"},
		{"1.1.1", greater, "1.1.0"},
		{"1.1.0", equal, "1.1.0"},
		{"1.1.0", greater, "0.2.0"},
		{"0.2.0", less, "0.3.0"},
		{"0.3.0", less, "1.1.1"},
		{"1.1.1", less, "1.2.0"},
		{"1.2.0", greater, "1.1.0"},
		{"1.1.0", less, "1.6.5"},
		{"1.6.5", greater, "1.1.0"},
		{"1.1.0", less, "1.4.2"},
		{"1.4.2", greater, "1.1.1"},
		{"1.1.1", less, "2.8.1"},
		{"2.8.1", greater, "1.2.0"},
		{"1.2.0", less, "4.1.0"},
		{"4.1.0", greater, "0.4.1"},
		{"0.4.1", less, "1.9.1"},
		{"1.9.1", less, "2.1.1"},
		{"2.1.1", greater, "1.4.1"},
		{"1.4.1", greater, "0.9.1-r1"},
		{"0.9.1-r1", greater, "0.8.1"},
		{"0.8.1", less, "1.2.1-r1"},
		{"1.2.1-r1", greater, "1.1.0"},
		{"1.1.0", less, "1.2.1"},
		{"1.2.1", greater, "1.1.0"},
		{"1.1.0", greater, "0.1.1"},
		{"0.1.1", less, "1.2.1"},
		{"1.2.1", less, "4.1.0"},
		{"4.1.0", greater, "0.2.1-r1"},
		{"0.2.1-r1", less, "1.1.0"},
		{"1.1.0", less, "2.7.11"},
		{"2.7.11", greater, "1.0.2-r6"},
		{"1.0.2-r6", greater, "1.0.2"},
		{"1.0.2", greater, "0.8"},
		{"0.8", less, "1.1.1-r4"},
		{"1.1.1-r4", less, "222"},
		{"222", greater, "1.0.1"},
		{"1.0.1", less, "1.2.12-r1"},
		{"1.2.12-r1", greater, "1.2.8"},
		{"1.2.8", less, "1.2.9.1-r1"},
		{"1.2.9.1-r1", greater, "1.2.9.1"},
		{"1.2.9.1", less, "2.31-r1"},
		{"2.31-r1", greater, "2.31"},
		{"2.31", greater, "1.2.3-r1"},
		{"1.2.3-r1", greater, "1.2.3"},
		{"1.2.3", less, "4.2.5"},
		{"4.2.5", less, "4.3.2-r2"},
		{"1.3-r0", less, "1.3.1-r0"},
		{"1.3_pre1-r1", less, "1.3.2"},
		{"1.0_p10-r0", greater, "1.0_p9-r0"},
		// FIXME(kaniini): Clarify whether this version test must actually pass.
		// {"0.1.0_alpha_pre2", less, "0.1.0_alpha"},
		{"1.0.0_pre20191002222144-r0", less, "1.0.0_pre20210530193627-r0"},
		{"1.2.3-r0", equal, "1.2.3-r0"},
		{"0.0_git20230331", less, "0.0_git20230508"},
		{"2.0.0", less, "2.0.6-r0"},
		{"6.4_p20231125-r0", greater, "6.4-r2"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("compare %s %s %s", tt.versionA, tt.expected, tt.versionB), func(t *testing.T) {
			verA, err := parseVersion(tt.versionA)
			require.NoError(t, err, "%q unexpected error", err)

			verB, err := parseVersion(tt.versionB)
			require.NoError(t, err, "%q unexpected error", err)

			result := compareVersions(verA, verB)
			require.Equalf(t, tt.expected, result, "comparison (%s %s %s) must be correct", tt.versionA, tt.expected, tt.versionB)
		})
	}
}

func TestResolveVersion(t *testing.T) {
	pinPackage := testNamedPackageFromVersionAndPin("2.1.0", "pinA")
	lowestPackage := testNamedPackageFromVersionAndPin("1.2.3-r0", "")
	pkgs := []*repositoryPackage{
		lowestPackage,
		testNamedPackageFromVersionAndPin("1.3.6-r0", ""),
		testNamedPackageFromVersionAndPin("1.2.8-r0", ""),
		testNamedPackageFromVersionAndPin("1.7.1-r0", ""),
		testNamedPackageFromVersionAndPin("1.7.1-r1", ""),
		testNamedPackageFromVersionAndPin("2.0.6-r0", ""),
		pinPackage,
	}
	tests := []struct {
		version     string
		compare     versionDependency
		pin         string
		installed   *RepositoryPackage
		want        string
		description string
	}{
		{"1.2.3-r0", versionEqual, "", nil, "1.2.3-r0", "exact version match"},
		{"1.2.3-r10000", versionEqual, "", nil, "", "exact version no match"},
		{"2.0.0", versionGreater, "", nil, "2.0.6-r0", "greater than version match"},
		{"2.0.0", versionGreaterEqual, "", nil, "2.0.6-r0", "greater than or equal to version match"},
		{"2.0.0", versionGreaterEqual, "", pinPackage.RepositoryPackage, "2.1.0", "greater than or equal to version match with pin preinstalled"},
		{"3.0.0", versionGreaterEqual, "", nil, "", "greater than or equal to version no match"},
		{"2.1.0", versionEqual, "", nil, "", "equal match but pinned"},
		{"2.1.0", versionEqual, "", pinPackage.RepositoryPackage, "2.1.0", "equal match but pinned yet already installed"},
		{"2.1.0", versionEqual, "pinA", nil, "2.1.0", "equal match and pin match"},
		{"", versionAny, "", nil, "2.0.6-r0", "no requirement should get highest version"},
		{"", versionAny, "", pinPackage.RepositoryPackage, pinPackage.Version, "no requirement should get highest version with pin, if installed"},
		{"", versionAny, "", lowestPackage.RepositoryPackage, lowestPackage.Version, "no requirement should get installed priority"},
		{"1.6", versionTilde, "", nil, "", "no match"},
		{"1.7", versionTilde, "", nil, "1.7.1-r1", "fits within"},
		{"1.7.1", versionTilde, "", nil, "1.7.1-r1", "fits within"},
		{"1.7.1-r2", versionTilde, "", nil, "", "no match"},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			pr := NewPkgResolver(context.Background(), []NamedIndex{})
			found := pr.filterPackages(pkgs, withVersion(tt.version, tt.compare), withPreferPin(tt.pin), withInstalledPackage(tt.installed))
			// add the existing in, if any
			existing := make(map[string]*RepositoryPackage)
			if tt.installed != nil {
				existing[tt.installed.Name] = tt.installed
			}
			pr.sortPackages(found, nil, "", existing, tt.pin)
			if tt.want == "" {
				require.Nil(t, found, "version resolver should not find a package")
			} else {
				require.NotNil(t, found, "version resolver should find a package")
				require.Equal(t, found[0].Version, tt.want, "version resolver gets correct version")
			}
		})
	}
}

func TestResolverPackageNameVersionPin(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
		dep     versionDependency
		pin     string
	}{
		{"agetty", "agetty", "", versionAny, ""},
		{"foo-dev", "foo-dev", "", versionAny, ""},
		{"name@edge", "name", "", versionAny, "edge"},
		{"name=1.2.3", "name", "1.2.3", versionEqual, ""},
		{"name>1.2.3", "name", "1.2.3", versionGreater, ""},
		{"name<1.2.3", "name", "1.2.3", versionLess, ""},
		{"name>=1.2.3", "name", "1.2.3", versionGreaterEqual, ""},
		{"name<=1.2.3", "name", "1.2.3", versionLessEqual, ""},
		{"name@edge=1.2.3", "name@edge=1.2.3", "", versionAny, ""}, // wrong order, so just returns the whole thing
		{"name=1.2.3@community", "name", "1.2.3", versionEqual, "community"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			stuff := resolvePackageNameVersionPin(tt.input)
			require.Equal(t, tt.name, stuff.name)
			require.Equal(t, tt.version, stuff.version)
			require.Equal(t, tt.dep, stuff.dep)
			require.Equal(t, tt.pin, stuff.pin)
		})
	}
}
