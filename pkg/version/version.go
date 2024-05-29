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

// Package version provides methods for *non-authoritative* version parsing and comparison.
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// versionRegex how to parse versions.
// see https://github.com/alpinelinux/apk-tools/blob/50ab589e9a5a84592ee4c0ac5a49506bb6c552fc/src/version.c#
// for information on pinning, see https://wiki.alpinelinux.org/wiki/Alpine_Package_Keeper#Repository_pinning
// To quote:
//
//   After which you can "pin" dependencies to these tags using:
//
//      apk add stableapp newapp@edge bleedingapp@testing
//   Apk will now by default only use the untagged repositories, but adding a tag to specific package:
//
//   1. will prefer the repository with that tag for the named package, even if a later version of the package is available in another repository
//
//   2. allows pulling in dependencies for the tagged package from the tagged repository (though it prefers to use untagged repositories to satisfy dependencies if possible)

var (
	versionRegex     = regexp.MustCompile(`^([0-9]+)((\.[0-9]+)*)([a-z]?)((_alpha|_beta|_pre|_rc)([0-9]*))?((_cvs|_svn|_git|_hg|_p)([0-9]*))?((-r)([0-9]+))?$`)
	packageNameRegex = regexp.MustCompile(`^([^@=><~]+)(([=><~]+)([^@]+))?(@([a-zA-Z0-9]+))?$`)
)

func init() {
	versionRegex.Longest()
	packageNameRegex.Longest()
}

type packageVersionPreModifier int
type packageVersionPostModifier int

// the order of these matters!
const (
	packageVersionPreModifierNone  packageVersionPreModifier = 0
	packageVersionPreModifierAlpha packageVersionPreModifier = 1
	packageVersionPreModifierBeta  packageVersionPreModifier = 2
	packageVersionPreModifierPre   packageVersionPreModifier = 3
	packageVersionPreModifierRC    packageVersionPreModifier = 4
	packageVersionPreModifierMax   packageVersionPreModifier = 1000
)
const (
	packageVersionPostModifierNone packageVersionPostModifier = 0
	packageVersionPostModifierCVS  packageVersionPostModifier = 1
	packageVersionPostModifierSVN  packageVersionPostModifier = 2
	packageVersionPostModifierGit  packageVersionPostModifier = 3
	packageVersionPostModifierHG   packageVersionPostModifier = 4
	packageVersionPostModifierP    packageVersionPostModifier = 5
	packageVersionPostModifierMax  packageVersionPostModifier = 1000
)

type Version struct {
	numbers          []int
	letter           rune
	preSuffix        packageVersionPreModifier
	preSuffixNumber  int
	postSuffix       packageVersionPostModifier
	postSuffixNumber int
	revision         int
}

// Parse parses a version string into a Version struct.
func Parse(version string) (*Version, error) {
	// TODO: Make this not use regex.
	parts := versionRegex.FindAllStringSubmatch(version, -1)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid version %s, could not parse", version)
	}
	actuals := parts[0]
	numbers := make([]int, 0, 10)
	if len(actuals) != 14 {
		return nil, fmt.Errorf("invalid version %s, could not find enough components", version)
	}

	// get the first version number
	num, err := strconv.Atoi(actuals[1])
	if err != nil {
		return nil, fmt.Errorf("invalid version %s, first part is not number: %w", version, err)
	}
	numbers = append(numbers, num)

	// get any other version numbers
	if actuals[2] != "" {
		subparts := strings.Split(actuals[2], ".")
		for i, s := range subparts {
			if s == "" {
				continue
			}
			num, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("invalid version %s, part %d is not number: %w", version, i, err)
			}
			numbers = append(numbers, num)
		}
	}
	var letter rune
	if len(actuals[4]) > 0 {
		letter = rune(actuals[4][0])
	}
	var preSuffix packageVersionPreModifier
	switch actuals[6] {
	case "_alpha":
		preSuffix = packageVersionPreModifierAlpha
	case "_beta":
		preSuffix = packageVersionPreModifierBeta
	case "_pre":
		preSuffix = packageVersionPreModifierPre
	case "_rc":
		preSuffix = packageVersionPreModifierRC
	case "":
		preSuffix = packageVersionPreModifierNone
	default:
		return nil, fmt.Errorf("invalid version %s, pre-suffix %s is not valid", version, actuals[6])
	}
	var preSuffixNumber int
	if actuals[7] != "" {
		num, err := strconv.Atoi(actuals[7])
		if err != nil {
			return nil, fmt.Errorf("invalid version %s, suffix %s number %s is not number: %w", version, actuals[6], actuals[7], err)
		}
		preSuffixNumber = num
	}

	var postSuffix packageVersionPostModifier
	switch actuals[9] {
	case "_cvs":
		postSuffix = packageVersionPostModifierCVS
	case "_svn":
		postSuffix = packageVersionPostModifierSVN
	case "_git":
		postSuffix = packageVersionPostModifierGit
	case "_hg":
		postSuffix = packageVersionPostModifierHG
	case "_p":
		postSuffix = packageVersionPostModifierP
	case "":
		postSuffix = packageVersionPostModifierNone
	default:
		return nil, fmt.Errorf("invalid version %s, suffix %s is not valid", version, actuals[9])
	}
	var postSuffixNumber int
	if actuals[10] != "" {
		num, err := strconv.Atoi(actuals[10])
		if err != nil {
			return nil, fmt.Errorf("invalid version %s, post-suffix %s number %s is not number: %w", version, actuals[9], actuals[10], err)
		}
		postSuffixNumber = num
	}

	var revision int
	if actuals[13] != "" {
		num, err := strconv.Atoi(actuals[13])
		if err != nil {
			return nil, fmt.Errorf("invalid version %s, revision %s is not number: %w", version, actuals[13], err)
		}
		revision = num
	}
	return &Version{
		numbers:          numbers,
		letter:           letter,
		preSuffix:        preSuffix,
		preSuffixNumber:  preSuffixNumber,
		postSuffix:       postSuffix,
		postSuffixNumber: postSuffixNumber,
		revision:         revision,
	}, nil
}

type versionCompare = int

const (
	greater versionCompare = 1
	equal   versionCompare = 0
	less    versionCompare = -1
)

// Compare compares versions based on https://dev.gentoo.org/~ulm/pms/head/pms.html#x1-250003.2
func Compare(actual, required Version) int {
	for i := 0; i < len(actual.numbers) && i < len(required.numbers); i++ {
		if actual.numbers[i] > required.numbers[i] {
			return greater
		}
		if actual.numbers[i] < required.numbers[i] {
			return less
		}
	}
	// if we made it here, the parts that were the same size are equal
	if len(actual.numbers) > len(required.numbers) {
		return greater
	}
	if len(actual.numbers) < len(required.numbers) {
		return less
	}
	// same length of numbers, same numbers
	// compare letters
	if actual.letter > required.letter {
		return greater
	}
	if actual.letter < required.letter {
		return less
	}
	// same letters
	// compare pre-suffixes
	// because None is 0 but the lowest priority to make it easy to have a sane default,
	// but lowest priority, we need some extra logic to handle
	actualPreSuffix, requiredPreSuffix := actual.preSuffix, required.preSuffix
	if actualPreSuffix == packageVersionPreModifierNone {
		actualPreSuffix = packageVersionPreModifierMax
	}
	if requiredPreSuffix == packageVersionPreModifierNone {
		requiredPreSuffix = packageVersionPreModifierMax
	}
	if actualPreSuffix > requiredPreSuffix {
		return greater
	}
	if actualPreSuffix < requiredPreSuffix {
		return less
	}
	// same pre-suffixes, compare pre-suffix numbers
	if actual.preSuffixNumber > required.preSuffixNumber {
		return greater
	}
	if actual.preSuffixNumber < required.preSuffixNumber {
		return less
	}
	// same pre-suffix numbers
	// compare post-suffixes
	//
	// Note that whereas we do a None -> Max transformation for pre-suffixes, we intentionally
	// leave post-suffixes alone, because they do not indicate a pre-release and should sort
	// greater than a version lacking a post-suffix.
	if actual.postSuffix > required.postSuffix {
		return greater
	}
	if actual.postSuffix < required.postSuffix {
		return less
	}
	// same post-suffixes, compare post-suffix numbers
	if actual.postSuffixNumber > required.postSuffixNumber {
		return greater
	}
	if actual.postSuffixNumber < required.postSuffixNumber {
		return less
	}
	// same post-suffix numbers
	// compare revisions
	if actual.revision > required.revision {
		return greater
	}
	if actual.revision < required.revision {
		return less
	}
	return equal
}
