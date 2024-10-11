package sse_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/replicate/replicate-go/internal/sse"
)

func TestDecodeOneEventNoSpace(t *testing.T) {
	input := `event:output
id:123abc
data:giraffe

`
	d := sse.NewDecoder(strings.NewReader(input))

	e, err := d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "123abc", e.ID)
	assert.Equal(t, "giraffe\n", e.Data)
}

func TestDecodeOneEventWithSpace(t *testing.T) {
	input := `event: output
id: 123abc
data:   giraffe

`
	d := sse.NewDecoder(strings.NewReader(input))

	e, err := d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "123abc", e.ID)
	// only one space should be trimmed
	assert.Equal(t, "  giraffe\n", e.Data)
}

func TestDecodeOneEventMultipleData(t *testing.T) {
	input := `event:output
data:giraffe
data:rhino
data:wombat

`
	d := sse.NewDecoder(strings.NewReader(input))

	e, err := d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "giraffe\nrhino\nwombat\n", e.Data)
}

func TestDecodeOneEventHugeData(t *testing.T) {
	// this test is mainly to make sure we're not constrained by the
	// bufio.Reader buffer size
	input := fmt.Sprintf(`event:output
data:%s

`, strings.Repeat("0123456789abcdef", 1_000_000))
	d := sse.NewDecoder(strings.NewReader(input))

	e, err := d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	// 16_000_000 data bytes and the terminal LF character
	assert.Equal(t, 16_000_001, len(e.Data))
}

func TestDecodeManyEvents(t *testing.T) {
	input := `event:output
id:alpha1
data:giraffe

event:output
id:bravo2
data:rhino

event:output
id:gamma3
data:pine marten

`
	d := sse.NewDecoder(strings.NewReader(input))

	e, err := d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "alpha1", e.ID)
	assert.Equal(t, "giraffe\n", e.Data)

	e, err = d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "bravo2", e.ID)
	assert.Equal(t, "rhino\n", e.Data)

	e, err = d.Next()

	require.NoError(t, err)

	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "gamma3", e.ID)
	assert.Equal(t, "pine marten\n", e.Data)
}

func TestDecodeEarlyEOF(t *testing.T) {
	input := ``
	d := sse.NewDecoder(strings.NewReader(input))

	_, err := d.Next()

	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
