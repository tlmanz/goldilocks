// Copyright 2019 FairwindsOps Inc
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

package dashboard

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Most dashboard responses are 100KB+ HTML; gzip cuts that 4–6x. We skip
// already-compressed binary asset types so we don't waste CPU re-encoding them.
var skipCompressionExt = map[string]struct{}{
	".png":   {},
	".jpg":   {},
	".jpeg":  {},
	".gif":   {},
	".webp":  {},
	".woff":  {},
	".woff2": {},
	".ico":   {},
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gz.Write(b)
}

// gzipMiddleware compresses responses when the client advertises gzip support
// and the request path doesn't look like a pre-compressed binary asset.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		if shouldSkipCompression(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() {
			_ = gz.Close()
			gzipWriterPool.Put(gz)
		}()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		// Once compressed the length header is wrong; drop it so downstream
		// proxies don't truncate the body.
		w.Header().Del("Content-Length")

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gz: gz}, r)
	})
}

func shouldSkipCompression(path string) bool {
	dot := strings.LastIndex(path, ".")
	if dot < 0 {
		return false
	}
	_, skip := skipCompressionExt[strings.ToLower(path[dot:])]
	return skip
}
