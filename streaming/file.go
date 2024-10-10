package streaming

import (
	"context"
	"io"
)

// File represents a file output from a model over an SSE stream.  On the wire,
// it might be a data URL or a regular http URL.  File abstracts over this and
// provides a way to get the data regardless of the implementation.
type File interface {
	// Body fetches the content of the file.  If there are any errors, the
	// io.ReadCloser will be nil.  It is the caller's responsibility to close
	// the io.ReadCloser.
	Body(ctx context.Context) (io.ReadCloser, error)
}
