package plugins

import (
	zipa "archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
	"gopkg.in/yaml.v3"
)

const (
	sourcesFolder  = "sources"
	archivesFolder = "archives"
	stateFilename  = "state.json"
	goPathSrc      = "src"
	pluginManifest = ".traefik.yml"
)

const pluginsURL = "https://plugins.traefik.io/public/"

const (
	hashHeader = "X-Plugin-Hash"
)

const verifiedHashSuffix = ".verified"

// ManagerOptions the options of a Traefik plugins manager.
type ManagerOptions struct {
	Output string
}

// Manager manages Traefik plugins lifecycle operations including storage, and manifest reading.
type Manager struct {
	downloader PluginDownloader

	stateFile string

	archives string
	sources  string
	goPath   string
}

// NewManager creates a new Traefik plugins manager.
func NewManager(downloader PluginDownloader, opts ManagerOptions) (*Manager, error) {
	sourcesRootPath := filepath.Join(filepath.FromSlash(opts.Output), sourcesFolder)
	err := resetDirectory(sourcesRootPath)
	if err != nil {
		return nil, err
	}

	goPath, err := os.MkdirTemp(sourcesRootPath, "gop-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create GoPath: %w", err)
	}

	archivesPath := filepath.Join(filepath.FromSlash(opts.Output), archivesFolder)
	err = os.MkdirAll(archivesPath, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create archives directory %s: %w", archivesPath, err)
	}

	return &Manager{
		downloader: downloader,
		stateFile:  filepath.Join(archivesPath, stateFilename),
		archives:   archivesPath,
		sources:    filepath.Join(goPath, goPathSrc),
		goPath:     goPath,
	}, nil
}

// InstallPlugin download and unzip the given plugin.
//
// When the registry is unavailable, the previously downloaded archive is reused,
// so that a Traefik restart during a registry outage does not disable every plugin.
// The cached archive is only used if its hash matches the pinned hash of the plugin,
// or the hash recorded the last time the registry validated it.
func (m *Manager) InstallPlugin(ctx context.Context, plugin Descriptor) error {
	hash, err := m.downloader.Download(ctx, plugin.ModuleName, plugin.Version)
	if err != nil {
		if !isRegistryUnavailable(err) {
			return fmt.Errorf("unable to download plugin %s: %w", plugin.ModuleName, err)
		}

		if cacheErr := m.checkCachedArchive(ctx, plugin); cacheErr != nil {
			return fmt.Errorf("unable to download plugin %s: %w", plugin.ModuleName, errors.Join(err, cacheErr))
		}

		log.Ctx(ctx).Warn().Err(err).
			Str("module", plugin.ModuleName).
			Str("version", plugin.Version).
			Msg("Plugin download failed, using the previously verified cached archive")
	} else {
		if plugin.Hash != "" {
			if plugin.Hash != hash {
				return fmt.Errorf("invalid hash for plugin %s, expected %s, got %s", plugin.ModuleName, plugin.Hash, hash)
			}
		} else {
			err = m.downloader.Check(ctx, plugin.ModuleName, plugin.Version, hash)
			if err != nil {
				return fmt.Errorf("unable to check archive integrity of the plugin %s: %w", plugin.ModuleName, err)
			}
		}

		verifiedHashPath := m.buildVerifiedHashPath(plugin.ModuleName, plugin.Version)
		if err = os.WriteFile(verifiedHashPath, []byte(hash), 0o600); err != nil {
			log.Ctx(ctx).Warn().Err(err).
				Str("module", plugin.ModuleName).
				Str("version", plugin.Version).
				Msg("Unable to record the verified plugin archive hash, the archive will not be reused if the registry is unavailable")
		}
	}

	if err = m.unzip(plugin.ModuleName, plugin.Version); err != nil {
		return fmt.Errorf("unable to unzip plugin %s: %w", plugin.ModuleName, err)
	}

	return nil
}

// GoPath gets the plugins GoPath.
func (m *Manager) GoPath() string {
	return m.goPath
}

// ReadManifest reads a plugin manifest.
func (m *Manager) ReadManifest(moduleName string) (*Manifest, error) {
	return ReadManifest(m.goPath, moduleName)
}

// ReadManifest reads a plugin manifest.
func ReadManifest(goPath, moduleName string) (*Manifest, error) {
	p := filepath.Join(goPath, goPathSrc, filepath.FromSlash(moduleName), pluginManifest)

	file, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open the plugin manifest %s: %w", p, err)
	}

	defer func() { _ = file.Close() }()

	m := &Manifest{}
	err = yaml.NewDecoder(file).Decode(m)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the plugin manifest %s: %w", p, err)
	}

	return m, nil
}

// CleanArchives cleans plugins archives.
func (m *Manager) CleanArchives(plugins map[string]Descriptor) error {
	if _, err := os.Stat(m.stateFile); os.IsNotExist(err) {
		return nil
	}

	stateFile, err := os.Open(m.stateFile)
	if err != nil {
		return fmt.Errorf("failed to open state file %s: %w", m.stateFile, err)
	}

	previous := make(map[string]string)
	err = json.NewDecoder(stateFile).Decode(&previous)
	if err != nil {
		return fmt.Errorf("failed to decode state file %s: %w", m.stateFile, err)
	}

	for pName, pVersion := range previous {
		for _, desc := range plugins {
			if desc.ModuleName == pName && desc.Version != pVersion {
				archivePath := m.buildArchivePath(pName, pVersion)
				if err = os.RemoveAll(archivePath); err != nil {
					return fmt.Errorf("failed to remove archive %s: %w", archivePath, err)
				}

				verifiedHashPath := m.buildVerifiedHashPath(pName, pVersion)
				if err = os.RemoveAll(verifiedHashPath); err != nil {
					return fmt.Errorf("failed to remove archive verified hash %s: %w", verifiedHashPath, err)
				}
			}
		}
	}

	return nil
}

// WriteState writes the plugins state files.
func (m *Manager) WriteState(plugins map[string]Descriptor) error {
	state := make(map[string]string)

	for _, descriptor := range plugins {
		state[descriptor.ModuleName] = descriptor.Version
	}

	mp, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal plugin state: %w", err)
	}

	return os.WriteFile(m.stateFile, mp, 0o600)
}

// ResetAll resets all plugins related directories.
func (m *Manager) ResetAll() error {
	if m.goPath == "" {
		return errors.New("goPath is empty")
	}

	err := resetDirectory(filepath.Join(m.goPath, ".."))
	if err != nil {
		return fmt.Errorf("unable to reset plugins GoPath directory %s: %w", m.goPath, err)
	}

	err = resetDirectory(m.archives)
	if err != nil {
		return fmt.Errorf("unable to reset plugins archives directory: %w", err)
	}

	return nil
}

// checkCachedArchive verifies that the cached archive of the plugin can be used without downloading it again.
func (m *Manager) checkCachedArchive(ctx context.Context, plugin Descriptor) error {
	hash, err := computeHash(m.buildArchivePath(plugin.ModuleName, plugin.Version))
	if err != nil {
		return fmt.Errorf("no usable cached archive: %w", err)
	}

	if plugin.Hash != "" {
		if plugin.Hash != hash {
			return fmt.Errorf("invalid hash for plugin %s, expected %s, got %s", plugin.ModuleName, plugin.Hash, hash)
		}

		return nil
	}

	verifiedHash, err := os.ReadFile(m.buildVerifiedHashPath(plugin.ModuleName, plugin.Version))
	if err != nil {
		return fmt.Errorf("cached archive was never verified: %w", err)
	}

	if subtle.ConstantTimeCompare(bytes.TrimSpace(verifiedHash), []byte(hash)) != 1 {
		return errors.New("cached archive does not match its verified hash")
	}

	// The validation endpoint may still be reachable, and an explicit rejection (e.g. a revoked version) must not be ignored.
	if err = m.downloader.Check(ctx, plugin.ModuleName, plugin.Version, hash); err != nil && !isRegistryUnavailable(err) {
		return fmt.Errorf("unable to check archive integrity of the plugin %s: %w", plugin.ModuleName, err)
	}

	return nil
}

func (m *Manager) unzip(pName, pVersion string) error {
	err := m.unzipModule(pName, pVersion)
	if err == nil {
		return nil
	}

	// Unzip as a generic archive if the module unzip fails.
	// This is useful for plugins that have vendor directories or other structures.
	// This is also useful for wasm plugins.
	return m.unzipArchive(pName, pVersion)
}

func (m *Manager) unzipModule(pName, pVersion string) error {
	src := m.buildArchivePath(pName, pVersion)
	dest := filepath.Join(m.sources, filepath.FromSlash(pName))

	return zip.Unzip(dest, module.Version{Path: pName, Version: pVersion}, src)
}

func (m *Manager) unzipArchive(pName, pVersion string) error {
	zipPath := m.buildArchivePath(pName, pVersion)

	archive, err := zipa.OpenReader(zipPath)
	if err != nil {
		return err
	}

	defer func() { _ = archive.Close() }()

	dest := filepath.Join(m.sources, filepath.FromSlash(pName))

	for _, f := range archive.File {
		err = m.unzipFile(f, dest)
		if err != nil {
			return fmt.Errorf("unable to unzip %s: %w", f.Name, err)
		}
	}

	return nil
}

func (m *Manager) unzipFile(f *zipa.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}

	defer func() { _ = rc.Close() }()

	// Split to discard the first part of the path when the archive is a Yaegi go plugin with vendoring.
	// In this case the path starts with `[organization]-[project]-[release commit sha1]/`.
	pathParts := strings.SplitN(f.Name, "/", 2)
	var fileName string
	if len(pathParts) < 2 {
		fileName = pathParts[0]
	} else {
		fileName = pathParts[1]
	}

	// Validate and sanitize the file path.
	cleanName := filepath.Clean(fileName)
	if strings.Contains(cleanName, "..") {
		return fmt.Errorf("invalid file path in archive: %s", f.Name)
	}

	filePath := filepath.Join(dest, cleanName)
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving file path: %w", err)
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolving destination directory: %w", err)
	}

	if !strings.HasPrefix(absFilePath, absDest) {
		return fmt.Errorf("file path escapes destination directory: %s", absFilePath)
	}

	if f.FileInfo().IsDir() {
		err = os.MkdirAll(filePath, f.Mode())
		if err != nil {
			return fmt.Errorf("unable to create archive directory %s: %w", filePath, err)
		}

		return nil
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0o750)
	if err != nil {
		return fmt.Errorf("unable to create archive directory %s for file %s: %w", filepath.Dir(filePath), filePath, err)
	}

	elt, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}

	defer func() { _ = elt.Close() }()

	_, err = io.Copy(elt, rc)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) buildArchivePath(pName, pVersion string) string {
	return filepath.Join(m.archives, filepath.FromSlash(pName), pVersion+".zip")
}

func (m *Manager) buildVerifiedHashPath(pName, pVersion string) string {
	return m.buildArchivePath(pName, pVersion) + verifiedHashSuffix
}

// isRegistryUnavailable reports whether the error comes from the registry being unreachable or failing,
// as opposed to the registry answering that the plugin version is unknown or invalid.
func isRegistryUnavailable(err error) bool {
	if statusErr, ok := errors.AsType[*registryStatusError](err); ok {
		return statusErr.StatusCode >= http.StatusInternalServerError || statusErr.StatusCode == http.StatusTooManyRequests
	}

	return true
}

func resetDirectory(dir string) error {
	dirPath, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("unable to get absolute path of %s: %w", dir, err)
	}

	currentPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get the current directory: %w", err)
	}

	if strings.HasPrefix(currentPath, dirPath) {
		return fmt.Errorf("cannot be deleted: the directory path %s is the parent of the current path %s", dirPath, currentPath)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("unable to remove directory %s: %w", dirPath, err)
	}

	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("unable to create directory %s: %w", dirPath, err)
	}

	return nil
}

func computeHash(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	sum := hash.Sum(nil)

	return hex.EncodeToString(sum), nil
}
