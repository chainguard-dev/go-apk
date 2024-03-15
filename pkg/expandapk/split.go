// Copyright 2024 Chainguard, Inc.
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

package expandapk

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/klauspost/compress/gzip"
	"go.opentelemetry.io/otel"
)

type SplitAPK struct {
	// The size in bytes of the entire apk (sum of all tar.gz file sizes)
	Size int64

	// Whether or not the apk contains a signature
	// Note: currently unused
	Signed bool

	// The package signature filename (a.k.a. ".SIGN...") in tar.gz format
	SignatureFile string

	// The control data filename (a.k.a. ".PKGINFO") in tar.gz format
	ControlFile string

	// The package data filename in .tar.gz format
	PackageFile string

	// The temporary parent directory containing all exploded .tar/.tar.gz contents
	tempDir string
}

func (a *SplitAPK) Close() error {
	return os.RemoveAll(a.tempDir)
}

// Split takes an APK stream and divides it into 2-3 files (signature, control, data).
// Callers are expected to Close() the SplitAPK to clean up these files.
// This is a stripped down version of [ExpandApk] that doesn't do expensive caching.
func Split(ctx context.Context, source io.Reader) (*SplitAPK, error) {
	_, span := otel.Tracer("go-apk").Start(ctx, "SplitAPK")
	defer span.End()

	dir, err := os.MkdirTemp("", "split-apk")
	if err != nil {
		return nil, err
	}

	gzipStreams := []string{}
	maxStreamsReached := false
	totalSize := int64(0)

	sw, err := newExpandApkWriter(dir, "stream", "tar.gz")
	if err != nil {
		return nil, err
	}

	exR := newExpandApkReader(source)

	tr := io.TeeReader(exR, sw)

	var gzi *gzip.Reader
	for {
		// Control section uses sha1.
		if err := sw.Next(); err != nil {
			if err == errExpandApkWriterMaxStreams {
				maxStreamsReached = true
				exR.EnableFastRead()
			} else {
				return nil, fmt.Errorf("expandApk error 5: %w", err)
			}
		}

		if gzi == nil {
			gzi, err = gzip.NewReader(tr)
		} else {
			err = gzi.Reset(tr)
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}

		if !maxStreamsReached {
			gzi.Multistream(false)
		}

		copied, err := io.Copy(io.Discard, gzi)
		if err != nil {
			return nil, fmt.Errorf("expandApk error 3: %w", err)
		}
		totalSize += copied

		gzipStreams = append(gzipStreams, sw.CurrentName())
	}

	if err := gzi.Close(); err != nil {
		return nil, fmt.Errorf("expandApk error 6: %w", err)
	}
	if err := sw.CloseFile(); err != nil {
		return nil, fmt.Errorf("expandApk error 7: %w", err)
	}

	numGzipStreams := len(gzipStreams)

	var signed bool
	var controlDataIndex int
	switch numGzipStreams {
	case 3:
		signed = true
		controlDataIndex = 1
	case 2:
		controlDataIndex = 0
	default:
		return nil, fmt.Errorf("invalid number of tar streams: %d", numGzipStreams)
	}

	split := SplitAPK{
		tempDir:     dir,
		Signed:      signed,
		Size:        totalSize,
		ControlFile: gzipStreams[controlDataIndex],
		PackageFile: gzipStreams[controlDataIndex+1],
	}
	if signed {
		split.SignatureFile = gzipStreams[0]
	}

	return &split, nil
}
