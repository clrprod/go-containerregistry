package registry

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type manifests struct {
	// maps repo -> manifest tag/digest -> manifest
	manifests map[string]map[string][]byte
	lock      sync.Mutex
}

func isManifest(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "manifests"
}

// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pulling-an-image-manifest
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pushing-an-image
func (m *manifests) handle(resp http.ResponseWriter, req *http.Request) *regError {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	target := elem[len(elem)-1]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	if req.Method == "GET" {
		m.lock.Lock()
		defer m.lock.Unlock()
		c, ok := m.manifests[repo]
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "NAME_UNKNOWN",
				Message: "Unknown name",
			}
		}
		m, ok := c[target]
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "MANIFEST_UNKNOWN",
				Message: "Unknown manifest",
			}
		}
		resp.Header().Set("Content-Length", fmt.Sprint(len(m)))
		resp.WriteHeader(http.StatusOK)
		io.Copy(resp, bytes.NewReader(m))
		return nil
	}

	if req.Method == "HEAD" {
		m.lock.Lock()
		defer m.lock.Unlock()
		if _, ok := m.manifests[repo]; !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "NAME_UNKNOWN",
				Message: "Unknown name",
			}
		}
		m, ok := m.manifests[repo][target]
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "MANIFEST_UNKNOWN",
				Message: "Unknown manifest",
			}
		}
		resp.Header().Set("Content-Length", fmt.Sprint(len(m)))
		resp.WriteHeader(http.StatusOK)
		return nil
	}

	if req.Method == "PUT" {
		m.lock.Lock()
		defer m.lock.Unlock()
		if _, ok := m.manifests[repo]; !ok {
			m.manifests[repo] = map[string][]byte{}
		}
		b := &bytes.Buffer{}
		io.Copy(b, req.Body)
		m.manifests[repo][target] = b.Bytes()
		resp.WriteHeader(http.StatusCreated)
		return nil
	}
	return &regError{
		Status:  http.StatusBadRequest,
		Code:    "METHOD_UNKNOWN",
		Message: "We don't understand your method + url",
	}
}
