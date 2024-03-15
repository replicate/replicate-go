package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
)

type File struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ContentType string            `json:"content_type"`
	Size        int               `json:"size"`
	Etag        string            `json:"etag"`
	Checksums   map[string]string `json:"checksums"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   string            `json:"created_at"`
	ExpiresAt   string            `json:"expires_at"`
	URLs        map[string]string `json:"urls"`

	rawJSON json.RawMessage `json:"-"`
}

var _ json.Unmarshaler = (*File)(nil)

func (f *File) UnmarshalJSON(data []byte) error {
	f.rawJSON = data
	type Alias File
	alias := &struct{ *Alias }{Alias: (*Alias)(f)}
	return json.Unmarshal(data, alias)
}

func (f *File) RawJSON() json.RawMessage {
	return f.rawJSON
}

type CreateFileOptions struct {
	Filename    string            `json:"filename"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateFileFromPath creates a new file from a file path.
func (r *Client) CreateFileFromPath(ctx context.Context, filePath string, options *CreateFileOptions) (*File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	if options == nil {
		options = &CreateFileOptions{}
	}

	if options.Filename == "" {
		_, options.Filename = filepath.Split(filePath)
	}

	if options.ContentType == "" {
		if options.Filename != "" {
			ext := filepath.Ext(options.Filename)
			options.ContentType = mime.TypeByExtension(ext)
		}
	}

	return r.createFile(ctx, f, *options)
}

// CreateFileFromBytes creates a new file from bytes.
func (r *Client) CreateFileFromBytes(ctx context.Context, data []byte, options *CreateFileOptions) (*File, error) {
	buf := bytes.NewBuffer(data)

	if options == nil {
		options = &CreateFileOptions{}
	}

	if options.ContentType == "" {
		options.ContentType = http.DetectContentType(data)
	}

	return r.createFile(ctx, buf, *options)
}

// CreateFileFromBuffer creates a new file from a buffer.
func (r *Client) CreateFileFromBuffer(ctx context.Context, buf *bytes.Buffer, options *CreateFileOptions) (*File, error) {
	if options == nil {
		options = &CreateFileOptions{}
	}

	return r.createFile(ctx, buf, *options)
}

// CreateFile creates a new file.
func (r *Client) createFile(ctx context.Context, reader io.Reader, options CreateFileOptions) (*File, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	filename := options.Filename
	if filename == "" {
		filename = "file"
	}

	contentType := options.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="content"; filename="%s"`, filename))
	h.Set("Content-Type", contentType)

	content, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(content, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write file to form: %w", err)
	}

	if options.Metadata != nil {
		metadata, err := json.Marshal(options.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		err = writer.WriteField("metadata", string(metadata))
		if err != nil {
			return nil, fmt.Errorf("failed to write metadata to form: %w", err)
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := r.newRequest(ctx, http.MethodPost, "/files", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	file := &File{}
	err = r.do(req, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

// ListFiles lists your files.
func (r *Client) ListFiles(ctx context.Context) (*Page[File], error) {
	response := &Page[File]{}
	err := r.fetch(ctx, http.MethodGet, "/files", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return response, nil
}

// GetFile retrieves information about a file.
func (r *Client) GetFile(ctx context.Context, fileID string) (*File, error) {
	file := &File{}
	err := r.fetch(ctx, http.MethodGet, fmt.Sprintf("/files/%s", fileID), nil, file)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

// DeleteFile deletes a file.
func (r *Client) DeleteFile(ctx context.Context, fileID string) error {
	err := r.fetch(ctx, http.MethodDelete, fmt.Sprintf("/files/%s", fileID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
