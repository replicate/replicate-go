package sse

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

type Event struct {
	Type string
	ID   string
	Data string
}

type Decoder struct {
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

var (
	eventField = []byte("event:")
	dataField  = []byte("data:")
	idField    = []byte("id:")
	retryField = []byte("retry:")
	space      = []byte{' '}
)

func buildEvent(t, id string, data *strings.Builder) Event {
	return Event{
		Type: t,
		ID:   id,
		Data: data.String(),
	}
}

func (d *Decoder) Next() (Event, error) {
	var t, id string
	var data strings.Builder
	for {
		line, err := d.r.ReadBytes('\n')
		if err == io.EOF {
			return buildEvent(t, id, &data), io.ErrUnexpectedEOF
		}
		if err != nil {
			return buildEvent(t, id, &data), err
		}

		switch {
		case line[0] == '\n':
			// a blank line finishes the event, so we return it
			return buildEvent(t, id, &data), nil
		case bytes.HasPrefix(line, eventField):
			t = string(bytes.TrimPrefix(line[6:len(line)-1], space))
		case bytes.HasPrefix(line, dataField):
			// strings.Builder.Write() always returns nil error, so we don't
			// need to handle it
			data.Write(bytes.TrimPrefix(line[5:], space))
		case bytes.HasPrefix(line, idField):
			id = string(bytes.TrimPrefix(line[3:len(line)-1], space))
		case bytes.HasPrefix(line, retryField):
		default:
			// ignore the line
		}

	}
}
