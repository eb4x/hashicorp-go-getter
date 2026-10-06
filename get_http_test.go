package getter

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	testing_helper "github.com/hashicorp/go-getter/v2/helper/testing"
)

func TestHttpGetter_impl(t *testing.T) {
	var _ Getter = new(HttpGetter)
}

func TestHttpGetter_header(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/header"

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_requestHeader(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	g.Header = make(http.Header)
	g.Header.Add("X-Foobar", "foobar")
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/expect-header"
	u.RawQuery = "expected=X-Foobar"

	req := &Request{
		Dst: dst,
		u:   &u,
	}

	// Get it!
	if err := g.GetFile(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, "Hello\n")
}

func TestHttpGetter_meta(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta"

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol:") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_metaSubdir(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta-subdir"

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "error downloading") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "sub.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_metaSubdirGlob(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta-subdir-glob"

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "error downloading") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "sub.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_none(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/none"

	req := &Request{
		Dst: dst,
		u:   &u,
	}

	// Get it!
	if err := g.Get(ctx, req); err == nil {
		t.Fatal("should error")
	}
}

func TestHttpGetter_resume(t *testing.T) {
	load := []byte(testHttpMetaStr)
	sha := sha256.New()
	if n, err := sha.Write(load); n != len(load) || err != nil {
		t.Fatalf("sha write failed: %d, %s", n, err)
	}
	checksum := hex.EncodeToString(sha.Sum(nil))
	downloadFrom := len(load) / 2

	ln := testHttpServer(t)
	defer ln.Close()

	dst := filepath.Join(t.TempDir(), "range")
	f, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if n, err := f.Write(load[:downloadFrom]); n != downloadFrom || err != nil {
		t.Fatalf("partial file write failed: %d, %s", n, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close failed: %s", err)
	}
	// Mark the partial file as coming from the same remote file.
	if err := os.Chtimes(dst, testHttpModTime, testHttpModTime); err != nil {
		t.Fatalf("chtimes failed: %s", err)
	}

	u := url.URL{
		Scheme:   "http",
		Host:     ln.Addr().String(),
		Path:     "/range",
		RawQuery: "checksum=" + checksum,
	}
	t.Logf("url: %s", u.String())
	ctx := context.Background()

	// Finish getting it!
	if _, err := GetFile(ctx, dst, u.String()); err != nil {
		t.Fatalf("finishing download should not error: %v", err)
	}

	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("readfile failed: %v", err)
	}

	if string(b) != string(load) {
		t.Fatalf("file differs: got:\n%s\n expected:\n%s\n", string(b), string(load))
	}

	// Get it again
	if _, err := GetFile(ctx, dst, u.String()); err != nil {
		t.Fatalf("should not error: %v", err)
	}
}

// The server may support Byte-Range, but has no size for the requested object
func TestHttpGetter_resumeNoRange(t *testing.T) {
	load := []byte(testHttpMetaStr)
	sha := sha256.New()
	if n, err := sha.Write(load); n != len(load) || err != nil {
		t.Fatalf("sha write failed: %d, %s", n, err)
	}
	checksum := hex.EncodeToString(sha.Sum(nil))
	downloadFrom := len(load) / 2

	ln := testHttpServer(t)
	defer ln.Close()

	dst := filepath.Join(t.TempDir(), "range")
	f, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if n, err := f.Write(load[:downloadFrom]); n != downloadFrom || err != nil {
		t.Fatalf("partial file write failed: %d, %s", n, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close failed: %s", err)
	}
	// Mark the partial file as coming from the same remote file.
	if err := os.Chtimes(dst, testHttpModTime, testHttpModTime); err != nil {
		t.Fatalf("chtimes failed: %s", err)
	}

	u := url.URL{
		Scheme:   "http",
		Host:     ln.Addr().String(),
		Path:     "/no-range",
		RawQuery: "checksum=" + checksum,
	}
	t.Logf("url: %s", u.String())
	ctx := context.Background()

	// Finish getting it!
	if _, err := GetFile(ctx, dst, u.String()); err != nil {
		t.Fatalf("finishing download should not error: %v", err)
	}

	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("readfile failed: %v", err)
	}

	if string(b) != string(load) {
		t.Fatalf("file differs: got:\n%s\n expected:\n%s\n", string(b), string(load))
	}
}

// An existing destination is only kept or resumed when its modification time
// matches the Last-Modified time of the remote file.
func TestHttpGetter_GetFileExisting(t *testing.T) {
	load := testHttpMetaStr
	staleTime := testHttpModTime.Add(-time.Hour)

	tc := []struct {
		name         string
		existing     string
		existingTime time.Time
		noHead       bool
		emptyHead    bool
		noGet        bool
		wantRange    string
		wantIfRange  string
	}{
		{name: "current", existing: load, existingTime: testHttpModTime, noGet: true},
		{name: "current-empty-head", existing: "", existingTime: testHttpModTime, emptyHead: true},
		{name: "current-prefix", existing: load[:10], existingTime: testHttpModTime, wantRange: "bytes=10-", wantIfRange: testHttpModTime.Format(http.TimeFormat)},
		{name: "stale-prefix", existing: "xxxxxxxxxx", existingTime: staleTime},
		{name: "stale-same-size", existing: strings.Repeat("x", len(load)), existingTime: staleTime},
		{name: "stale-larger", existing: load + "trailing", existingTime: staleTime},
		{name: "current-larger", existing: load + "trailing", existingTime: testHttpModTime},
		{name: "no-head-larger", existing: load + "trailing", existingTime: testHttpModTime, noHead: true},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var gets []http.Header
			dst := testHttpExistingFile(t, tt.existing, tt.existingTime)
			req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					mu.Lock()
					gets = append(gets, r.Header.Clone())
					mu.Unlock()
				}
				if tt.emptyHead && r.Method == "HEAD" {
					// Some servers report no size for HEAD requests (#538).
					w.Header().Set("Accept-Ranges", "bytes")
					w.Header().Set("Last-Modified", testHttpModTime.Format(http.TimeFormat))
					w.Header().Set("Content-Length", "0")
					return
				}
				http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(load))
			})

			g := &HttpGetter{DoNotCheckHeadFirst: tt.noHead}
			if err := g.GetFile(context.Background(), req); err != nil {
				t.Fatalf("err: %s", err)
			}
			testing_helper.AssertContents(t, dst, load)
			testHttpAssertModTime(t, dst, testHttpModTime)

			mu.Lock()
			defer mu.Unlock()
			wantGets := 1
			if tt.noGet {
				wantGets = 0
			}
			if len(gets) != wantGets {
				t.Fatalf("got %d GET requests, want %d", len(gets), wantGets)
			}
			if wantGets == 1 && (gets[0].Get("Range") != tt.wantRange || gets[0].Get("If-Range") != tt.wantIfRange) {
				t.Fatalf("bad GET request: got Range %q, If-Range %q, want %q, %q",
					gets[0].Get("Range"), gets[0].Get("If-Range"), tt.wantRange, tt.wantIfRange)
			}
		})
	}
}

// A server that ignores the range request sends the whole file, which
// replaces the existing content.
func TestHttpGetter_GetFileRangeIgnored(t *testing.T) {
	load := testHttpMetaStr
	dst := testHttpExistingFile(t, load[:10], testHttpModTime)
	req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Last-Modified", testHttpModTime.Format(http.TimeFormat))
		w.Header().Set("Content-Length", strconv.Itoa(len(load)))
		if r.Method == "GET" {
			w.Write([]byte(load))
		}
	})

	if err := new(HttpGetter).GetFile(context.Background(), req); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, load)
}

// A partial response must continue the existing file up to the end of the
// remote file, and come from the remote file that the existing bytes came
// from. A server may also send the whole file as partial content. A rejected
// response drops the existing bytes, so that the next call starts over.
func TestHttpGetter_GetFilePartialContent(t *testing.T) {
	load := testHttpMetaStr
	n := len(load)
	rest := fmt.Sprintf("bytes 10-%d/%d", n-1, n)
	otherTime := testHttpModTime.Add(time.Hour).Format(http.TimeFormat)

	tc := []struct {
		name         string
		existing     string
		contentRange string
		body         string
		lastModified string
		date         string
		chunked      bool
		errExpected  bool
		stamped      bool
	}{
		{name: "resume", existing: load[:10], contentRange: rest, body: load[10:], stamped: true},
		{name: "resume-same-last-modified", existing: load[:10], contentRange: rest, body: load[10:], lastModified: testHttpModTime.Format(http.TimeFormat), stamped: true},
		{name: "resume-chunked-long", existing: load[:10], contentRange: rest, body: load[10:] + "xxx", chunked: true, stamped: true},
		{name: "whole-file", contentRange: fmt.Sprintf("bytes 0-%d/%d", n-1, n), body: load},
		{name: "wrong-start", existing: load[:10], contentRange: fmt.Sprintf("bytes 5-%d/%d", n-1, n), body: load[5:], errExpected: true},
		{name: "short", existing: load[:10], contentRange: fmt.Sprintf("bytes 10-%d/%d", n-2, n), body: load[10 : n-1], errExpected: true},
		{name: "long", existing: load[:10], contentRange: rest, body: load[10:] + "xxx", errExpected: true},
		{name: "wrong-size", existing: load[:10], contentRange: fmt.Sprintf("bytes 10-%d/%d", n, n+1), body: load[10:] + "x", errExpected: true},
		{name: "unknown-size", existing: load[:10], contentRange: fmt.Sprintf("bytes 10-%d/*", n-1), body: load[10:], errExpected: true},
		{name: "other-last-modified", existing: load[:10], contentRange: rest, body: load[10:], lastModified: otherTime, errExpected: true},
		{name: "weak-last-modified", existing: load[:10], contentRange: rest, body: load[10:], lastModified: otherTime, date: otherTime, errExpected: true},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			dst := testHttpExistingFile(t, tt.existing, testHttpModTime)
			req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "HEAD" {
					http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(load))
					return
				}
				if tt.lastModified != "" {
					w.Header().Set("Last-Modified", tt.lastModified)
				}
				if tt.date != "" {
					w.Header().Set("Date", tt.date)
				}
				w.Header().Set("Content-Range", tt.contentRange)
				if !tt.chunked {
					w.Header().Set("Content-Length", strconv.Itoa(len(tt.body)))
				}
				w.WriteHeader(http.StatusPartialContent)
				if tt.chunked {
					// Send the headers without a Content-Length.
					w.(http.Flusher).Flush()
				}
				w.Write([]byte(tt.body))
			})

			err := new(HttpGetter).GetFile(context.Background(), req)
			if tt.errExpected != (err != nil) {
				t.Fatalf("err: %v", err)
			}
			if tt.errExpected {
				testing_helper.AssertContents(t, dst, "")
			} else {
				testing_helper.AssertContents(t, dst, load)
			}
			if tt.stamped {
				testHttpAssertModTime(t, dst, testHttpModTime)
			} else {
				testHttpAssertNotModTime(t, dst, testHttpModTime)
			}
		})
	}
}

// After a rejected partial response, the next call downloads the whole file.
func TestHttpGetter_GetFilePartialContentRetry(t *testing.T) {
	load := testHttpMetaStr
	dst := testHttpExistingFile(t, load[:10], testHttpModTime)
	req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			// Answer every range request with the whole file.
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(load)-1, len(load)))
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte(load))
			return
		}
		http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(load))
	})

	g := new(HttpGetter)
	if err := g.GetFile(context.Background(), req); err == nil {
		t.Fatal("expected error")
	}
	if err := g.GetFile(context.Background(), req); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, load)
	testHttpAssertModTime(t, dst, testHttpModTime)
}

// A remote file that changes between the HEAD and GET requests is downloaded
// again. The destination gets the earlier Last-Modified time of the HEAD
// request, so the next call, whose HEAD request reports the new time,
// downloads it once more.
func TestHttpGetter_GetFileChanged(t *testing.T) {
	oldLoad := testHttpMetaStr
	newLoad := strings.ToUpper(testHttpMetaStr)
	newModTime := testHttpModTime.Add(time.Hour)

	dst := testHttpExistingFile(t, oldLoad[:10], testHttpModTime)
	req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(oldLoad))
			return
		}
		http.ServeContent(w, r, "file", newModTime, strings.NewReader(newLoad))
	})

	if err := new(HttpGetter).GetFile(context.Background(), req); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, newLoad)
	testHttpAssertModTime(t, dst, testHttpModTime)
}

// The GET request may be redirected to another server (a CDN) that reports
// another Last-Modified time for the same file than the HEAD request. The
// destination gets the earlier time: a later GET time must not cause a
// download on every call, and an earlier one (a stale cache) must not be kept.
func TestHttpGetter_GetFileLastModifiedDiffers(t *testing.T) {
	load := testHttpMetaStr

	tc := []struct {
		name        string
		getModTime  time.Time
		wantModTime time.Time
		wantGets    int
	}{
		{name: "later-get", getModTime: testHttpModTime.Add(3 * time.Second), wantModTime: testHttpModTime, wantGets: 1},
		{name: "earlier-get", getModTime: testHttpModTime.Add(-time.Hour), wantModTime: testHttpModTime.Add(-time.Hour), wantGets: 2},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var gets int
			dst := filepath.Join(t.TempDir(), "file")
			req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "HEAD" {
					http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(load))
					return
				}
				mu.Lock()
				gets++
				mu.Unlock()
				http.ServeContent(w, r, "file", tt.getModTime, strings.NewReader(load))
			})

			g := new(HttpGetter)
			for i := 0; i < 2; i++ {
				if err := g.GetFile(context.Background(), req); err != nil {
					t.Fatalf("err: %s", err)
				}
			}
			testing_helper.AssertContents(t, dst, load)
			testHttpAssertModTime(t, dst, tt.wantModTime)

			mu.Lock()
			defer mu.Unlock()
			if gets != tt.wantGets {
				t.Fatalf("got %d GET requests, want %d", gets, tt.wantGets)
			}
		})
	}
}

// A body that the Transport decompressed differs from the remote bytes, so it
// is not marked as (a prefix of) the remote file.
func TestHttpGetter_GetFileGzip(t *testing.T) {
	load := testHttpMetaStr
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write([]byte(load)); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("err: %s", err)
	}

	dst := filepath.Join(t.TempDir(), "file")
	req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		http.ServeContent(w, r, "file", testHttpModTime, bytes.NewReader(gz.Bytes()))
	})

	if err := new(HttpGetter).GetFile(context.Background(), req); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, load)
	testHttpAssertNotModTime(t, dst, testHttpModTime)
}

// An interrupted download keeps the Last-Modified time of the remote file, so
// that the next call resumes it.
func TestHttpGetter_GetFileInterrupted(t *testing.T) {
	load := testHttpMetaStr
	dst := filepath.Join(t.TempDir(), "file")
	req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.Header.Get("Range") == "" {
			// Send the first 10 bytes and drop the connection. Flush, so that
			// the client receives them before the abort.
			w.Header().Set("Last-Modified", testHttpModTime.Format(http.TimeFormat))
			w.Header().Set("Content-Length", strconv.Itoa(len(load)))
			w.Write([]byte(load[:10]))
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}
		http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(load))
	})

	g := new(HttpGetter)
	if err := g.GetFile(context.Background(), req); err == nil {
		t.Fatal("expected error")
	}
	testing_helper.AssertContents(t, dst, load[:10])
	testHttpAssertModTime(t, dst, testHttpModTime)

	if err := g.GetFile(context.Background(), req); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, load)
}

// Without a Last-Modified time that is a strong validator, an existing
// destination cannot be matched to the remote file, so it is downloaded again
// and its modification time is not set.
func TestHttpGetter_GetFileWeakLastModified(t *testing.T) {
	load := testHttpMetaStr

	tc := []struct {
		name         string
		lastModified time.Time
		date         time.Time
		noDate       bool
	}{
		{name: "none"},
		{name: "same-second", lastModified: testHttpModTime, date: testHttpModTime},
		{name: "no-date", lastModified: testHttpModTime, noDate: true},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			dst := testHttpExistingFile(t, load[:10], testHttpModTime)
			req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
				// The existing file must not be resumed.
				if r.Header.Get("Range") != "" {
					http.Error(w, "unexpected range", http.StatusBadRequest)
					return
				}
				if !tt.date.IsZero() {
					w.Header().Set("Date", tt.date.Format(http.TimeFormat))
				}
				if tt.noDate {
					w.Header()["Date"] = nil
				}
				http.ServeContent(w, r, "file", tt.lastModified, strings.NewReader(load))
			})

			if err := new(HttpGetter).GetFile(context.Background(), req); err != nil {
				t.Fatalf("err: %s", err)
			}
			testing_helper.AssertContents(t, dst, load)
			fi, err := os.Stat(dst)
			if err != nil {
				t.Fatal(err)
			}
			if fi.ModTime().Equal(testHttpModTime) {
				t.Fatal("modification time set from a weak Last-Modified")
			}
		})
	}
}

func TestHttpGetter_file(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempTestFile(t)
	defer os.RemoveAll(filepath.Dir(dst))

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/file"

	req := &Request{
		Dst: dst,
		u:   &u,
	}

	// Get it!
	if err := g.GetFile(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("err: %s", err)
	}
	testing_helper.AssertContents(t, dst, "Hello\n")
}

func TestHttpGetter_auth(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta-auth"
	u.User = url.UserPassword("foo", "bar")

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol:") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_authNetrc(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta"

	// Write the netrc file
	path, closer := testing_helper.TempFileWithContent(t, fmt.Sprintf(testHttpNetrc, ln.Addr().String()))
	defer closer()
	defer tempEnv(t, "NETRC", path)()

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol:") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// test round tripper that only returns an error
type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("test round tripper")
}

// verify that the default httpClient no longer comes from http.DefaultClient
func TestHttpGetter_cleanhttp(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()

	// break the default http client
	http.DefaultClient.Transport = errRoundTripper{}
	defer func() {
		http.DefaultClient.Transport = http.DefaultTransport
	}()
	ctx := context.Background()

	g := new(HttpGetter)
	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/header"

	req := &Request{
		Dst:     dst,
		Src:     u.String(),
		u:       &u,
		GetMode: ModeDir,
	}

	// Get it, which should error because it uses the file protocol.
	err := g.Get(ctx, req)
	if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol:") {
		t.Fatalf("unexpected error: %v", err)
	}
	// But, using a wrapper client with a file getter will work.
	c := &Client{
		Getters: []Getter{
			g,
			new(FileGetter),
		},
	}

	if _, err = c.Get(ctx, req); err != nil {
		t.Fatalf("err: %s", err)
	}

}

func TestHttpGetter__RespectsContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ln := testHttpServer(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/file"

	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	rt := hookableHTTPRoundTripper{
		before: func(req *http.Request) {
			err := req.Context().Err()
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Expected http.Request with canceled.Context, got: %v", err)
			}
		},
		RoundTripper: http.DefaultTransport,
	}

	g := new(HttpGetter)
	g.Client = &http.Client{
		Transport: &rt,
	}

	req := Request{
		Dst: dst,
		u:   &u,
	}
	err := g.Get(ctx, &req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestHttpGetter__XTerraformGetLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := testHttpServerWithXTerraformGetLoop(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/loop"

	dst := testing_helper.TempDir(t)
	defer os.RemoveAll(dst)

	g := new(HttpGetter)
	g.XTerraformGetLimit = 10
	g.Client = &http.Client{}

	req := Request{
		Dst:     dst,
		u:       &u,
		GetMode: ModeDir,
	}

	err := g.Get(ctx, &req)
	if !strings.Contains(err.Error(), "too many X-Terraform-Get redirects") {
		t.Fatalf("too many X-Terraform-Get redirects, got: %v", err)
	}
}

func TestHttpGetter__XTerraformGetDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := testHttpServerWithXTerraformGetLoop(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/loop"
	dst := testing_helper.TempDir(t)

	g := new(HttpGetter)
	g.XTerraformGetDisabled = true
	g.Client = &http.Client{}

	req := Request{
		Dst:     dst,
		u:       &u,
		GetMode: ModeDir,
	}

	err := g.Get(ctx, &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestHttpGetter__XTerraformGetProxyBypass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := testHttpServerWithXTerraformGetProxyBypass(t)

	proxyLn := testHttpServerProxy(t, ln.Addr().String())

	t.Logf("starting malicious server on: %v", ln.Addr().String())
	t.Logf("starting proxy on: %v", proxyLn.Addr().String())

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/start"
	dst := testing_helper.TempDir(t)

	proxy, err := url.Parse(fmt.Sprintf("http://%s/", proxyLn.Addr().String()))
	if err != nil {
		t.Fatalf("failed to parse proxy URL: %v", err)
	}

	transport := cleanhttp.DefaultTransport()
	transport.Proxy = http.ProxyURL(proxy)

	g := new(HttpGetter)
	g.XTerraformGetLimit = 10
	g.Client = &http.Client{
		Transport: transport,
	}

	client := &Client{
		Getters: []Getter{g},
	}

	req := Request{
		Dst: dst,
		Src: u.String(),
	}

	_, err = client.Get(ctx, &req)
	if err != nil {
		t.Logf("client get error: %v", err)
	}
}

func TestHttpGetter__XTerraformGetConfiguredGettersBypass(t *testing.T) {
	tc := []struct {
		name              string
		configuredGetters []Getter
		errExpected       bool
	}{
		{name: "configured getter for git protocol switch", configuredGetters: []Getter{new(GitGetter)}, errExpected: false},
		{name: "configured getter for multiple protocol switch", configuredGetters: []Getter{new(GitGetter), new(HgGetter), new(FileGetter)}, errExpected: false},
		{name: "configured getter for file protocol switch", configuredGetters: []Getter{new(FileGetter)}, errExpected: true},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ln := testHttpServerWithXTerraformGetConfiguredGettersBypass(t)

			var u url.URL
			u.Scheme = "http"
			u.Host = ln.Addr().String()
			u.Path = "/start"

			dst := testing_helper.TempDir(t)

			rt := hookableHTTPRoundTripper{
				before: func(req *http.Request) {
					t.Logf("making request")
				},
				RoundTripper: http.DefaultTransport,
			}

			g := new(HttpGetter)
			g.XTerraformGetLimit = 10
			g.Client = &http.Client{
				Transport: &rt,
			}

			client := &Client{
				Getters: []Getter{g},
			}
			client.Getters = append(client.Getters, tt.configuredGetters...)

			t.Logf("%v", u.String())

			req := Request{
				Dst:     dst,
				Src:     u.String(),
				GetMode: ModeDir,
			}

			_, err := client.Get(ctx, &req)
			// For configured getters that support git, the git repository doesn't exist so error will not be nil.
			// If we get a nil error when we expect one other than the git error git exited with -1 we should fail.
			if tt.errExpected && err == nil {
				t.Fatalf("error expected")
			}
			// We only care about the error messages that indicate that we can download the git header URL
			if tt.errExpected && err != nil {
				if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol:") {
					t.Fatalf("expected no getter available for X-Terraform-Get source protocol:, got: %v", err)
				}
			}
		})
	}
}

func TestHttpGetter__endless_body(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := testHttpServerWithEndlessBody(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/"
	dst := testing_helper.TempDir(t)

	g := new(HttpGetter)
	g.MaxBytes = 10
	g.DoNotCheckHeadFirst = true

	client := &Client{
		Getters: []Getter{g},
	}

	t.Logf("%v", u.String())

	req := Request{
		Dst:     dst,
		Src:     u.String(),
		GetMode: ModeFile,
	}

	_, err := client.Get(ctx, &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// MaxBytes also limits the download when a ProgressListener is set.
func TestHttpGetter_GetFileMaxBytesProgress(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "file")
	req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat(".", 100)))
	})
	p := &MockProgressTracking{}
	req.ProgressListener = p

	g := &HttpGetter{MaxBytes: 10, DoNotCheckHeadFirst: true}
	err := g.GetFile(context.Background(), req)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("err: %v", err)
	}
	testing_helper.AssertContents(t, dst, strings.Repeat(".", 10))
	if p.downloaded["file"] != 1 {
		t.Fatalf("progress tracked %d times, want 1", p.downloaded["file"])
	}
}

// A download that MaxBytes cut short is not resumed, so repeated calls cannot
// grow it past MaxBytes. Without a Content-Length, the cut is not an error.
func TestHttpGetter_GetFileMaxBytesResume(t *testing.T) {
	load := strings.Repeat(".", 100)

	tc := []struct {
		name    string
		chunked bool
	}{
		{name: "content-length"},
		{name: "chunked", chunked: true},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "file")
			req := testHttpFileRequest(t, dst, func(w http.ResponseWriter, r *http.Request) {
				if tt.chunked && r.Method == "GET" && r.Header.Get("Range") == "" {
					w.Header().Set("Last-Modified", testHttpModTime.Format(http.TimeFormat))
					w.WriteHeader(http.StatusOK)
					// Send the headers without a Content-Length.
					w.(http.Flusher).Flush()
					w.Write([]byte(load))
					return
				}
				http.ServeContent(w, r, "file", testHttpModTime, strings.NewReader(load))
			})

			g := &HttpGetter{MaxBytes: 10}
			for i := 0; i < 2; i++ {
				err := g.GetFile(context.Background(), req)
				if err != nil && !errors.Is(err, io.ErrShortWrite) {
					t.Fatalf("err: %s", err)
				}
				testing_helper.AssertContents(t, dst, load[:10])
			}
		})
	}
}

func TestHttpGetter_subdirLink(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln := testHttpServerSubDir(t)
	defer ln.Close()

	dst := t.TempDir()

	t.Logf("dst: %q", dst)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/regular-subdir//meta-subdir"

	g := new(HttpGetter)
	client := &Client{
		Getters: []Getter{g},
	}

	t.Logf("url: %q", u.String())

	req := Request{
		Dst:     dst,
		Src:     u.String(),
		GetMode: ModeAny,
	}

	_, err := client.Get(ctx, &req)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
}

func testHttpServerWithXTerraformGetLoop(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	header := fmt.Sprintf("http://%v%v", ln.Addr().String(), "/loop")

	mux := http.NewServeMux()
	mux.HandleFunc("/loop", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Terraform-Get", header)
		t.Logf("serving loop")
	})

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpServerWithXTerraformGetProxyBypass(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	header := fmt.Sprintf("http://%v/bypass", ln.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/start/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Terraform-Get", header)
		t.Logf("serving start")
	})

	mux.HandleFunc("/bypass", func(w http.ResponseWriter, r *http.Request) {
		t.Fail()
		t.Logf("bypassed proxy")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("serving HTTP server path: %v", r.URL.Path)
	})

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpServerWithXTerraformGetConfiguredGettersBypass(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	header := fmt.Sprintf("git::http://%v/some/repository.git", ln.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Terraform-Get", header)
		t.Logf("serving start")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("serving git HTTP server path: %v", r.URL.Path)
	})

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func TestHttpGetter_XTerraformWithClientFromContext(t *testing.T) {
	tc := []struct {
		name        string
		client      *Client
		errExpected bool
	}{
		{
			name: "default getters",
			client: &Client{
				Getters: Getters,
			},
			errExpected: false,
		},
		{
			name: "client configured with needed getters",
			client: &Client{
				Getters: []Getter{
					new(HttpGetter),
					new(FileGetter),
				},
			},
			errExpected: false,
		},
		{
			name:        "nil client",
			errExpected: true,
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ln := testHttpServer(t)
			defer ln.Close()
			ctx := context.Background()

			g := new(HttpGetter)
			dst := testing_helper.TempDir(t)
			defer os.RemoveAll(dst)

			var u url.URL
			u.Scheme = "http"
			u.Host = ln.Addr().String()
			u.Path = "/header"

			req := &Request{
				Dst:     dst,
				Src:     u.String(),
				u:       &u,
				GetMode: ModeDir,
			}

			// Using a client stored in the ctx with a file getter should work
			ctx = NewContextWithClient(ctx, tt.client)

			err := g.Get(ctx, req)
			if tt.errExpected && err == nil {
				t.Fatalf("error expected")
			}

			if err != nil {
				if !strings.Contains(err.Error(), "no getter available for X-Terraform-Get source protocol:") {
					t.Fatalf("expected no getter available for X-Terraform-Get source protocol:, got: %v", err)
				}
				return
			}

			// Verify the main file exists
			mainPath := filepath.Join(dst, "main.tf")
			if _, err := os.Stat(mainPath); err != nil {
				t.Fatalf("err: %s", err)
			}
		})
	}
}

func testHttpServerProxy(t *testing.T, upstreamHost string) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("serving proxy: %v: %#+v", r.URL.Path, r.Header)
		// create the reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(r.URL)
		// Note that ServeHttp is non blocking & uses a go routine under the hood
		proxy.ServeHTTP(w, r)
	})

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpServer(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/expect-header", testHttpHandlerExpectHeader)
	mux.HandleFunc("/file", testHttpHandlerFile)
	mux.HandleFunc("/header", testHttpHandlerHeader)
	mux.HandleFunc("/meta", testHttpHandlerMeta)
	mux.HandleFunc("/meta-auth", testHttpHandlerMetaAuth)
	mux.HandleFunc("/meta-subdir", testHttpHandlerMetaSubdir)
	mux.HandleFunc("/meta-subdir-glob", testHttpHandlerMetaSubdirGlob)
	mux.HandleFunc("/range", testHttpHandlerRange)
	mux.HandleFunc("/no-range", testHttpHandlerNoRange)

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpServerWithEndlessBody(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		for {
			w.Write([]byte(".\n"))
		}
	})

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpHandlerExpectHeader(w http.ResponseWriter, r *http.Request) {
	if expected, ok := r.URL.Query()["expected"]; ok {
		if r.Header.Get(expected[0]) != "" {
			w.Write([]byte("Hello\n"))
			return
		}
	}

	w.WriteHeader(400)
}

func testHttpHandlerFile(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello\n"))
}

func testHttpHandlerHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Terraform-Get", testModuleURL("basic").String())
	w.WriteHeader(200)
}

func testHttpHandlerMeta(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(testHttpMetaStr, testModuleURL("basic").String())))
}

func testHttpHandlerMetaAuth(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		w.WriteHeader(401)
		return
	}

	if user != "foo" || pass != "bar" {
		w.WriteHeader(401)
		return
	}

	w.Write([]byte(fmt.Sprintf(testHttpMetaStr, testModuleURL("basic").String())))
}

func testHttpHandlerMetaSubdir(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(testHttpMetaStr, testModuleURL("basic//subdir").String())))
}

func testHttpHandlerMetaSubdirGlob(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(testHttpMetaStr, testModuleURL("basic//sub*").String())))
}

func testHttpHandlerNone(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(testHttpNoneStr))
}

func testHttpHandlerRange(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && r.Header.Get("Range") == "" {
		http.Error(w, "range expected", http.StatusBadRequest)
		return
	}
	http.ServeContent(w, r, "range", testHttpModTime, strings.NewReader(testHttpMetaStr))
}

func testHttpHandlerNoRange(w http.ResponseWriter, r *http.Request) {
	load := []byte(testHttpMetaStr)
	switch r.Method {
	case "HEAD":
		// we support range, but the object size isn't known
		w.Header().Add("accept-ranges", "bytes")
		w.Header().Set("Last-Modified", testHttpModTime.Format(http.TimeFormat))
	default:
		if r.Header.Get("Range") != "" {
			http.Error(w, "range not supported", http.StatusBadRequest)
		}
		w.Write(load)
	}
}

func testHttpServerSubDir(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			t.Logf("serving: %v: %v: %#+[1]v", r.Method, r.URL.String(), r.Header)
		}
	})

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

// testHttpModTime is the Last-Modified time of the files served in tests.
var testHttpModTime = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

// testHttpFileRequest serves handler until the test ends, and returns a
// request that downloads from it to dst.
func testHttpFileRequest(t *testing.T, dst string, handler http.HandlerFunc) *Request {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Request{
		Dst: dst,
		u:   &url.URL{Scheme: "http", Host: server.Listener.Addr().String(), Path: "/file"},
	}
}

func testHttpExistingFile(t *testing.T, content string, modTime time.Time) string {
	t.Helper()

	dst := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(dst, []byte(content), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chtimes(dst, modTime, modTime); err != nil {
		t.Fatalf("err: %s", err)
	}
	return dst
}

func testHttpAssertModTime(t *testing.T, path string, want time.Time) {
	t.Helper()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !fi.ModTime().Equal(want) {
		t.Fatalf("bad modification time: got %s, want %s", fi.ModTime().UTC(), want.UTC())
	}
}

func testHttpAssertNotModTime(t *testing.T, path string, notWant time.Time) {
	t.Helper()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if fi.ModTime().Equal(notWant) {
		t.Fatalf("modification time should not be %s", notWant.UTC())
	}
}

const testHttpMetaStr = `
<html>
<head>
<meta name="terraform-get" content="%s">
</head>
</html>
`

const testHttpNoneStr = `
<html>
<head>
</head>
</html>
`

const testHttpNetrc = `
machine %s
login foo
password bar
`

type hookableHTTPRoundTripper struct {
	before func(req *http.Request)
	http.RoundTripper
}

func (m *hookableHTTPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.before != nil {
		m.before(req)
	}
	return m.RoundTripper.RoundTrip(req)
}
