package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// Keychain returns the combined credential chain: the Docker keychain
// (~/.docker/config.json incl. credential helpers) plus the Podman auth
// files ($XDG_RUNTIME_DIR/containers/auth.json, ~/.config/containers/auth.json).
func Keychain() authn.Keychain {
	return authn.NewMultiKeychain(authn.DefaultKeychain, podmanKeychain{})
}

// podmanKeychain resolves credentials from Podman's auth.json files.
type podmanKeychain struct{}

func (podmanKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	for _, path := range podmanAuthPaths() {
		a := readAuthFile(path, target.RegistryStr())
		if a != nil {
			return a, nil
		}
	}
	return authn.Anonymous, nil
}

func podmanAuthPaths() []string {
	var paths []string
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "containers", "auth.json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "containers", "auth.json"))
	}
	return paths
}

type authFile struct {
	Auths map[string]struct {
		Auth          string `json:"auth"`
		IdentityToken string `json:"identitytoken"`
	} `json:"auths"`
}

func readAuthFile(path, registry string) authn.Authenticator {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f authFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil
	}
	for host, entry := range f.Auths {
		if !registryMatches(host, registry) {
			continue
		}
		cfg := authn.AuthConfig{IdentityToken: entry.IdentityToken}
		if entry.Auth != "" {
			decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
			if err != nil {
				continue
			}
			user, pass, ok := strings.Cut(string(decoded), ":")
			if !ok {
				continue
			}
			cfg.Username, cfg.Password = user, pass
		}
		return authn.FromConfig(cfg)
	}
	return nil
}

// registryMatches handles the various ways docker.io can be recorded.
func registryMatches(entryHost, registry string) bool {
	h := strings.TrimPrefix(entryHost, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimSuffix(h, "/")
	if h == "index.docker.io/v1" {
		h = "index.docker.io"
	}
	if h == "docker.io" {
		h = "index.docker.io"
	}
	return h == registry
}

// AuthHeader builds the base64 X-Registry-Auth value for engine pulls,
// resolving credentials for imageRef's registry. Empty string = anonymous.
func AuthHeader(imageRef string) (string, error) {
	ref, err := name.ParseReference(imageRef, name.WeakValidation)
	if err != nil {
		return "", nil // unparsable: let the engine try anonymously
	}
	auth, err := Keychain().Resolve(ref.Context().Registry)
	if err != nil {
		return "", nil
	}
	cfg, err := auth.Authorization()
	if err != nil || (cfg.Username == "" && cfg.IdentityToken == "") {
		return "", nil
	}
	payload, err := json.Marshal(map[string]string{
		"username":      cfg.Username,
		"password":      cfg.Password,
		"identitytoken": cfg.IdentityToken,
		"serveraddress": ref.Context().RegistryStr(),
	})
	if err != nil {
		return "", fmt.Errorf("encode registry auth: %w", err)
	}
	return base64.URLEncoding.EncodeToString(payload), nil
}
