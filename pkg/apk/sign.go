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
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/psanford/memfs"

	"github.com/chainguard-dev/go-apk/pkg/tarball"
)

var (
	errNoPemBlock    = errors.New("no PEM block found")
	errDigestNotSHA1 = errors.New("digest is not a SHA1 hash")
	errNoPassphrase  = errors.New("key is encrypted but no passphrase was provided")
	errNoRSAKey      = errors.New("key is not an RSA key")
)

// Signer is responsible for signing the digest of some data and returning the signature.
type Signer interface {
	// Sign signs the given digest of some contents, and returns the signature.
	Sign(ctx context.Context, digest []byte) ([]byte, error)

	// KeyName returns the name of the key used to sign the data.
	KeyName() string
}

// Verifier is responsible for verifying a signature against a digest.
type Verifier interface {
	// Verify verifies the given signature against the given digest.
	Verify(ctx context.Context, digest, signature []byte) error
}

type SignerVerifier interface {
	Signer
	Verifier
}

// NewKeySignerVerifier returns a SignerVerifier that uses the given private key file to sign.
func NewKeySignerVerifier(privkeyFile, passphrase string) (SignerVerifier, error) {
	pub, err := os.ReadFile(privkeyFile + ".pub")
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}
	priv, err := os.ReadFile(privkeyFile)
	if err != nil {
		return nil, fmt.Errorf("read private key file: %w", err)
	}
	return &keySignerVerifier{
		pubKey:      pub,
		privKey:     priv,
		privkeyFile: privkeyFile,
		passphrase:  passphrase,
	}, nil
}

// NewKeyVerifier returns a Verifier that uses the given public key to verify.
func NewKeyVerifier(pubkey []byte) Verifier {
	return &keySignerVerifier{
		pubKey: pubkey,
	}
}

type keySignerVerifier struct {
	privkeyFile     string
	privKey, pubKey []byte
	passphrase      string
}

func (s *keySignerVerifier) KeyName() string {
	return filepath.Base(s.privkeyFile)
}

func (s *keySignerVerifier) Sign(_ context.Context, digest []byte) ([]byte, error) {
	if len(digest) != sha1.Size {
		return nil, errDigestNotSHA1
	}

	block, _ := pem.Decode(s.privKey)
	if block == nil {
		return nil, errNoPemBlock
	}

	blockData := block.Bytes
	if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck
		if s.passphrase == "" {
			return nil, errNoPassphrase
		}

		var decryptedBlockData []byte

		decryptedBlockData, err := x509.DecryptPEMBlock(block, []byte(s.passphrase)) //nolint:staticcheck
		if err != nil {
			return nil, fmt.Errorf("decrypt private key PEM block: %w", err)
		}

		blockData = decryptedBlockData
	}

	priv, err := x509.ParsePKCS1PrivateKey(blockData)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
	}

	return priv.Sign(rand.Reader, digest, crypto.SHA1)
}

func (s *keySignerVerifier) Verify(_ context.Context, digest, signature []byte) error {
	if len(digest) != sha1.Size {
		return errDigestNotSHA1
	}

	block, _ := pem.Decode(s.pubKey)
	if block == nil {
		return errNoPemBlock
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse PKIX public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return errNoRSAKey
	}

	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA1, digest, signature)
}

func SignIndex(logger *log.Logger, signer Signer, indexFile string) error {
	is, err := indexIsAlreadySigned(indexFile)
	if err != nil {
		return err
	}
	if is {
		logger.Printf("index %s is already signed, doing nothing", indexFile)
		return nil
	}

	logger.Printf("signing index %s with key %s", indexFile, signer.KeyName())

	indexData, indexDigest, err := ReadAndHashIndexFile(indexFile)
	if err != nil {
		return err
	}

	sigData, err := signer.Sign(context.Background(), indexDigest)
	if err != nil {
		return fmt.Errorf("unable to sign index: %w", err)
	}

	logger.Printf("appending signature to index %s", indexFile)

	sigFS := memfs.New()
	if err := sigFS.WriteFile(fmt.Sprintf(".SIGN.RSA.%s.pub", signer.KeyName()), sigData, 0644); err != nil {
		return fmt.Errorf("unable to append signature: %w", err)
	}

	// prepare control.tar.gz
	multitarctx, err := tarball.NewContext(
		tarball.WithSkipClose(true),
	)
	if err != nil {
		return fmt.Errorf("unable to build tarball context: %w", err)
	}

	logger.Printf("writing signed index to %s", indexFile)

	var sigBuffer bytes.Buffer
	if err := multitarctx.WriteArchive(&sigBuffer, sigFS); err != nil {
		return fmt.Errorf("unable to write signature tarball: %w", err)
	}

	idx, err := os.Create(indexFile)
	if err != nil {
		return fmt.Errorf("unable to open index for writing: %w", err)
	}
	defer idx.Close()

	if _, err := io.Copy(idx, &sigBuffer); err != nil {
		return fmt.Errorf("unable to write index signature: %w", err)
	}

	if _, err := idx.Write(indexData); err != nil {
		return fmt.Errorf("unable to write index data: %w", err)
	}

	logger.Printf("signed index %s with key %s", indexFile, signer.KeyName())

	return nil
}

func indexIsAlreadySigned(indexFile string) (bool, error) {
	index, err := os.Open(indexFile)
	if err != nil {
		return false, fmt.Errorf("cannot open index file %s: %w", indexFile, err)
	}
	defer index.Close()

	gzi, err := gzip.NewReader(index)
	if err != nil {
		return false, fmt.Errorf("cannot open index file %s as gzip: %w", indexFile, err)
	}
	defer gzi.Close()

	tari := tar.NewReader(gzi)
	for {
		hdr, err := tari.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false, fmt.Errorf("cannot read tar index %s: %w", indexFile, err)
		}

		if strings.HasPrefix(hdr.Name, ".SIGN.RSA") {
			return true, nil
		}
	}

	return false, nil
}

func ReadAndHashIndexFile(indexFile string) ([]byte, []byte, error) {
	indexBuf, err := os.ReadFile(indexFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read index for signing: %w", err)
	}

	indexDigest, err := HashData(indexBuf)

	return indexBuf, indexDigest, err
}

func HashData(data []byte) ([]byte, error) {
	digest := sha1.New() //nolint:gosec
	if n, err := digest.Write(data); err != nil || n != len(data) {
		return nil, fmt.Errorf("unable to hash data: %w", err)
	}
	return digest.Sum(nil), nil
}
