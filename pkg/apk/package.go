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
	"archive/tar"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chainguard-dev/go-apk/pkg/expandapk"
	"gopkg.in/ini.v1"
)

// PackageToIndex takes a Package and returns it as the string representation of lines in an index file.
//
// TODO(jason): This should take our Package type, and not alpine-go's. This requires pulling in a fork of RepositoryPackage.
func PackageToIndex(pkg *Package) (out []string) {
	out = append(out, fmt.Sprintf("P:%s", pkg.Name))
	out = append(out, fmt.Sprintf("V:%s", pkg.Version))
	out = append(out, fmt.Sprintf("A:%s", pkg.Arch))
	out = append(out, fmt.Sprintf("L:%s", pkg.License))
	out = append(out, fmt.Sprintf("T:%s", pkg.Description))
	out = append(out, fmt.Sprintf("o:%s", pkg.Origin))
	out = append(out, fmt.Sprintf("m:%s", pkg.Maintainer))
	out = append(out, fmt.Sprintf("U:%s", pkg.URL))
	out = append(out, fmt.Sprintf("D:%s", strings.Join(pkg.Dependencies, " ")))
	out = append(out, fmt.Sprintf("p:%s", strings.Join(pkg.Provides, " ")))
	out = append(out, fmt.Sprintf("c:%s", pkg.RepoCommit))
	out = append(out, fmt.Sprintf("i:%s", pkg.InstallIf))
	out = append(out, fmt.Sprintf("t:%d", pkg.BuildTime.Unix()))
	out = append(out, fmt.Sprintf("S:%d", pkg.Size))
	out = append(out, fmt.Sprintf("I:%d", pkg.InstalledSize))
	out = append(out, fmt.Sprintf("k:%d", pkg.ProviderPriority))
	if len(pkg.Checksum) > 0 {
		out = append(out, fmt.Sprintf("C:Q1%s", base64.StdEncoding.EncodeToString(pkg.Checksum)))
	}

	return
}

// Package represents a single package with the information present in an
// APKINDEX.
type Package struct {
	Name             string `ini:"pkgname"`
	Version          string `ini:"pkgver"`
	Arch             string `ini:"arch"`
	Description      string `ini:"pkgdesc"`
	License          string `ini:"license"`
	Origin           string `ini:"origin"`
	Maintainer       string `ini:"maintainer"`
	URL              string `ini:"url"`
	Checksum         []byte
	Dependencies     []string `ini:"depend,,allowshadow"`
	Provides         []string `ini:"provides,,allowshadow"`
	InstallIf        []string
	Size             uint64 `ini:"size"`
	InstalledSize    uint64
	ProviderPriority uint64
	BuildTime        time.Time
	BuildDate        int64  `ini:"builddate"`
	RepoCommit       string `ini:"commit"`
	Replaces         string `ini:"replaces"`
}

// Returns the package filename as it's named in a repository.
func (p *Package) Filename() string {
	return fmt.Sprintf("%s-%s.apk", p.Name, p.Version)
}

// ChecksumString returns a human-readable version of the checksum.
func (p *Package) ChecksumString() string {
	return "Q1" + base64.StdEncoding.EncodeToString(p.Checksum)
}

// ParsePackage parses a .apk file and returns a Package struct
func ParsePackage(ctx context.Context, apkPackage io.Reader) (*Package, error) {
	expanded, err := expandapk.ExpandApk(ctx, apkPackage, "")
	if err != nil {
		return nil, fmt.Errorf("expandApk(): %v", err)
	}

	control, err := expanded.ControlData()
	if err != nil {
		return nil, fmt.Errorf("expanded.ControlData(): %v", err)
	}
	tarRead := tar.NewReader(control)
	if _, err = tarRead.Next(); err != nil {
		return nil, fmt.Errorf("tarRead.Next(): %v", err)
	}

	cfg, err := ini.ShadowLoad(tarRead)
	if err != nil {
		return nil, fmt.Errorf("ini.ShadowLoad(): %w", err)
	}

	pkg := new(Package)
	if err = cfg.MapTo(pkg); err != nil {
		return nil, fmt.Errorf("cfg.MapTo(): %w", err)
	}
	pkg.BuildTime = time.Unix(pkg.BuildDate, 0).UTC()

	return pkg, nil
}
