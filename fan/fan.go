package fan

import (
	"bytes"
	"io"
	"sync"
)

// Reader wraps an io.Reader, allowing "clones" to be created via the View method.
type Reader struct {
	io.Reader

	s   []byte     // buffered io.Reader data
	mux sync.Mutex // could maybe be replaced by an RWMutex
}

// View returns a new io.Reader that behaves like a copy of the original io.Reader
func (r *Reader) View() io.Reader {
	var i int64 // current reading index
	return readFunc(func(p []byte) (int, error) {
		r.mux.Lock()
		defer r.mux.Unlock()
		// Declare the returned error here. It is only assigned by calls to
		// r.Reader.Read (`if` block below). That way callers see the io.EOF
		// error only when they reach the limit of r.s
		var err error
		// If the client has asked for more data than is available, we need to
		// grow the buffer.
		if i+int64(len(p)) > int64(len(r.s)) {
			cp := make([]byte, len(p))
			var n int // don't shadow err
			n, err = r.Reader.Read(cp)
			r.s = append(r.s, cp[:n]...)
		}
		n := copy(p, r.s[i:])
		i += int64(n)
		return n, err
	})
}

// Original returns the "original" io.Reader without any thread-safety working
// behind the scenes.
func (r *Reader) Original() io.Reader {
	return io.MultiReader(bytes.NewReader(r.s), r.Reader)
}

// ReadFunc follows the design of http.HandlerFunc, allowing us to create io.Reader
// functions that can exploit closured variables
type readFunc func(p []byte) (n int, err error)

func (rf readFunc) Read(p []byte) (n int, err error) { return rf(p) }
