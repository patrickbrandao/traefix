package plugins

import (
	"archive/zip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPPluginDownloader_Download(t *testing.T) {
	tests := []struct {
		name              string
		serverResponse    func(w http.ResponseWriter, r *http.Request)
		fileAlreadyExists bool
		expectError       bool
	}{
		{
			name: "successful download",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/zip")
				w.WriteHeader(http.StatusOK)

				require.NoError(t, fillDummyZip(w))
			},
		},
		{
			name: "not modified response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "", http.StatusNotModified)
			},
			fileAlreadyExists: true,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(test.serverResponse))
			defer server.Close()

			tempDir := t.TempDir()
			archivesPath := filepath.Join(tempDir, "archives")

			if test.fileAlreadyExists {
				createDummyZip(t, archivesPath)
			}

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			downloader := &RegistryDownloader{
				httpClient: server.Client(),
				baseURL:    baseURL,
				archives:   archivesPath,
			}

			ctx := t.Context()
			hash, err := downloader.Download(ctx, "test/plugin", "v1.0.0")

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)

				// Check if archive file was created
				archivePath := downloader.buildArchivePath("test/plugin", "v1.0.0")
				assert.FileExists(t, archivePath)
			}
		})
	}
}

func TestHTTPPluginDownloader_Check(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    require.ErrorAssertionFunc
	}{
		{
			name: "successful check",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectError: require.NoError,
		},
		{
			name: "failed check",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectError: require.Error,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(test.serverResponse))
			defer server.Close()

			tempDir := t.TempDir()
			archivesPath := filepath.Join(tempDir, "archives")

			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			downloader := &RegistryDownloader{
				httpClient: server.Client(),
				baseURL:    baseURL,
				archives:   archivesPath,
			}

			ctx := t.Context()

			err = downloader.Check(ctx, "test/plugin", "v1.0.0", "testhash")
			test.expectError(t, err)
		})
	}
}

func TestHTTPPluginDownloader_Download_interruptedKeepsCachedArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
	}))
	defer server.Close()

	archivesPath := filepath.Join(t.TempDir(), "archives")
	createDummyZip(t, archivesPath)

	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	downloader := &RegistryDownloader{
		httpClient: server.Client(),
		baseURL:    baseURL,
		archives:   archivesPath,
	}

	archivePath := downloader.buildArchivePath("test/plugin", "v1.0.0")
	cachedHash, err := computeHash(archivePath)
	require.NoError(t, err)

	_, err = downloader.Download(t.Context(), "test/plugin", "v1.0.0")
	require.Error(t, err)

	hash, err := computeHash(archivePath)
	require.NoError(t, err)
	assert.Equal(t, cachedHash, hash)

	entries, err := os.ReadDir(filepath.Dir(archivePath))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestHTTPPluginDownloader_statusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	downloader := &RegistryDownloader{
		httpClient: server.Client(),
		baseURL:    baseURL,
		archives:   filepath.Join(t.TempDir(), "archives"),
	}

	_, err = downloader.Download(t.Context(), "test/plugin", "v1.0.0")
	var statusErr *registryStatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
	assert.False(t, isRegistryUnavailable(err))

	err = downloader.Check(t.Context(), "test/plugin", "v1.0.0", "hash")
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusNotFound, statusErr.StatusCode)
}

func createDummyZip(t *testing.T, path string) {
	t.Helper()

	err := os.MkdirAll(path+"/test/plugin/", 0o755)
	require.NoError(t, err)

	zipfile, err := os.Create(path + "/test/plugin/v1.0.0.zip")
	require.NoError(t, err)
	defer zipfile.Close()

	err = fillDummyZip(zipfile)
	require.NoError(t, err)
}

func fillDummyZip(w io.Writer) error {
	writer := zip.NewWriter(w)

	file, err := writer.Create("test.txt")
	if err != nil {
		return err
	}

	_, _ = file.Write([]byte("test content"))
	_ = writer.Close()
	return nil
}
