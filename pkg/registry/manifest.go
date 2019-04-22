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
	// maps container -> manifest tag/ digest -> manifest
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
func (m *manifests) handle(resp http.ResponseWriter, req *http.Request) {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	target := elem[len(elem)-1]
	container := strings.Join(elem[1:len(elem)-2], "/")

	if req.Method == "GET" {
		m.lock.Lock()
		defer m.lock.Unlock()
		if _, ok := m.manifests[container]; !ok {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		m, ok := m.manifests[container][target]
		if !ok {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Header().Set("Content-Length", fmt.Sprint(len(m)))
		resp.WriteHeader(http.StatusOK)
		io.Copy(resp, bytes.NewReader(m))
		return
	}

	if req.Method == "HEAD" {
		m.lock.Lock()
		defer m.lock.Unlock()
		if _, ok := m.manifests[container]; !ok {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		m, ok := m.manifests[container][target]
		if !ok {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
		resp.Header().Set("Content-Length", fmt.Sprint(len(m)))
		resp.WriteHeader(http.StatusOK)
		return
	}

	if req.Method == "PUT" {
		m.lock.Lock()
		defer m.lock.Unlock()
		if _, ok := m.manifests[container]; !ok {
			m.manifests[container] = map[string][]byte{}
		}
		b := &bytes.Buffer{}
		io.Copy(b, req.Body)
		m.manifests[container][target] = b.Bytes()
		resp.WriteHeader(http.StatusCreated)
		return
	}
	resp.WriteHeader(http.StatusBadRequest)
}
