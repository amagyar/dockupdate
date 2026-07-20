// Package registry checks whether running images have updates available by
// comparing local repo digests against remote registry manifest digests.
// Checks are HEAD requests only: no image data is downloaded.
package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Kind classifies the outcome of an update check.
type Kind int

const (
	// KindUnknown is the zero value: no check result yet. It must remain
	// first so an unchecked Result is never mistaken for a real outcome.
	KindUnknown Kind = iota
	KindUpdateAvailable
	KindUpToDate
	KindLocalBuild // locally built image, no repo digest to compare
	KindPinned     // image pinned by digest
	KindFailed     // check could not complete (registry error, auth, ...)
)

// Result is the outcome of checking one image reference.
type Result struct {
	Kind         Kind
	RemoteDigest string // remote manifest digest observed during the check
	Err          error
}

func (r Result) String() string {
	switch r.Kind {
	case KindUnknown:
		return "unknown"
	case KindUpdateAvailable:
		return "update available"
	case KindUpToDate:
		return "up to date"
	case KindLocalBuild:
		return "local build"
	case KindPinned:
		return "pinned by digest"
	case KindFailed:
		if r.Err != nil {
			return "check failed: " + r.Err.Error()
		}
		return "check failed"
	}
	return "unknown"
}

// Checker performs digest checks against registries.
type Checker struct {
	nameOpts   []name.Option
	remoteOpts []remote.Option
	platform   ggcrv1.Platform
}

// NewChecker builds a Checker authenticating with the Docker and Podman
// keychains. platform selects the image variant when a tag points to a
// multi-arch index. insecure allows plain HTTP (used in tests).
func NewChecker(platform ggcrv1.Platform, insecure bool) *Checker {
	c := &Checker{platform: platform}
	if insecure {
		c.nameOpts = append(c.nameOpts, name.Insecure)
	}
	c.remoteOpts = append(c.remoteOpts, remote.WithAuthFromKeychain(Keychain()))
	if platform.OS != "" {
		c.remoteOpts = append(c.remoteOpts, remote.WithPlatform(platform))
	}
	return c
}

// IsPinned reports whether the reference is pinned by digest.
func IsPinned(imageRef string) bool {
	return strings.Contains(imageRef, "@sha256:")
}

// Check compares the local repo digests of imageRef against the remote
// manifest digest(s). An update is available when the remote digest differs
// from every local repo digest.
func (c *Checker) Check(ctx context.Context, imageRef string, localDigests []string) Result {
	if IsPinned(imageRef) {
		return Result{Kind: KindPinned}
	}
	if len(localDigests) == 0 {
		return Result{Kind: KindLocalBuild}
	}
	ref, err := name.ParseReference(imageRef, c.nameOpts...)
	if err != nil {
		return Result{Kind: KindFailed, Err: fmt.Errorf("parse reference: %w", err)}
	}

	candidates, primary, err := c.remoteDigests(ctx, ref)
	if err != nil {
		return Result{Kind: KindFailed, Err: err}
	}
	for _, local := range localDigests {
		for _, remote := range candidates {
			if local == remote {
				return Result{Kind: KindUpToDate, RemoteDigest: primary}
			}
		}
	}
	return Result{Kind: KindUpdateAvailable, RemoteDigest: primary}
}

// remoteDigests returns the set of digests that count as "current": the tag
// digest plus, for multi-arch indexes, the platform-specific manifest digest
// (engines record one or the other depending on their image store).
func (c *Checker) remoteDigests(ctx context.Context, ref name.Reference) (candidates []string, primary string, err error) {
	opts := append([]remote.Option{remote.WithContext(ctx)}, c.remoteOpts...)
	head, err := remote.Head(ref, opts...)
	if err != nil {
		return nil, "", fmt.Errorf("query registry: %w", err)
	}
	primary = head.Digest.String()
	candidates = append(candidates, primary)

	if head.MediaType.IsIndex() {
		img, err := remote.Image(ref, opts...)
		if err == nil {
			if d, derr := img.Digest(); derr == nil {
				candidates = append(candidates, d.String())
			}
		}
	} else if head.MediaType == types.DockerManifestSchema1 {
		// Legacy schema; nothing more to add.
	}
	return candidates, primary, nil
}

// PlatformFromEngine maps engine info to a registry platform.
func PlatformFromEngine(osType, arch string) ggcrv1.Platform {
	if osType == "" {
		osType = "linux"
	}
	if arch == "" {
		arch = "amd64"
	}
	return ggcrv1.Platform{OS: osType, Architecture: arch}
}

// ensure authn import is used (keychain wiring lives in keychain.go).
var _ = authn.Anonymous
