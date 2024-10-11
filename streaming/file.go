package streaming

import (
	"context"
	"io"
)

// FileStreamer represents a stream of output files from a model.
type FileStreamer interface {
	io.Closer
	NextFile(ctx context.Context) (File, error)
}

// File represents a file output from a model over an SSE stream.  On the wire,
// it might be a data URL or a regular http URL.  File abstracts over this and
// provides a way to get the data regardless of the implementation.
type File interface {
	// Body fetches the content of the file.  If there are any errors, the
	// io.ReadCloser will be nil.  It is the caller's responsibility to close
	// the io.ReadCloser.
	Body(ctx context.Context) (io.ReadCloser, error)
}
