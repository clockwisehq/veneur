package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/veneur/ssf"
)

func TestReadSSFStream(t *testing.T) {
	msg := &ssf.SSFSpan{
		Version:        1,
		TraceId:        1,
		Id:             2,
		ParentId:       3,
		StartTimestamp: 9000,
		EndTimestamp:   9001,
		Tags:           map[string]string{},
	}
	commonTags := map[string]string{"gloobles": "toots"}
	// Write it to a buffer twice:
	buf := bytes.NewBuffer([]byte{})
	_, err := WriteSSF(buf, msg)
	require.NoError(t, err)
	_, err = WriteSSF(buf, msg)
	require.NoError(t, err)
	// Read the first frame:
	{
		span, err := ReadSSF(buf, commonTags)
		require.NoError(t, err)
		assert.Equal(t, int32(1), span.Version)
		assert.Equal(t, int64(1), span.TraceId)
		assert.Equal(t, int64(2), span.Id)
		assert.Equal(t, int64(3), span.ParentId)
		assert.Equal(t, int64(9000), span.GetStartTimestamp())
		assert.Equal(t, int64(9001), span.GetEndTimestamp())

		assert.Equal(t, "toots", span.Tags["gloobles"])
	}
	// Read the second frame:
	{
		span, err := ReadSSF(buf, commonTags)
		require.NoError(t, err)
		assert.Equal(t, int32(1), span.Version)
		assert.Equal(t, int64(1), span.TraceId)
		assert.Equal(t, int64(2), span.Id)
		assert.Equal(t, int64(3), span.ParentId)
		assert.Equal(t, int64(9000), span.GetStartTimestamp())
		assert.Equal(t, int64(9001), span.GetEndTimestamp())
		assert.Equal(t, "toots", span.Tags["gloobles"])
	}
}

func TestEOF(t *testing.T) {
	msg := &ssf.SSFSpan{
		Version:        1,
		TraceId:        1,
		Id:             2,
		ParentId:       3,
		StartTimestamp: 9000,
		EndTimestamp:   9001,
		Tags:           map[string]string{},
	}
	buf := bytes.NewBuffer([]byte{})
	_, err := WriteSSF(buf, msg)
	require.NoError(t, err)
	// First frame should work:
	{
		read, err := ReadSSF(buf, nil)
		if assert.NoError(t, err) {
			assert.NotNil(t, read)
		}
	}
	// Second frame should return a plain EOF error:
	{
		read, err := ReadSSF(buf, nil)
		assert.False(t, IsFramingError(err))
		if assert.Equal(t, io.EOF, err) {
			assert.Nil(t, read)
		}
	}
	// subsequent reads should get EOF too:
	{
		read, err := ReadSSF(buf, nil)
		assert.False(t, IsFramingError(err))
		if assert.Equal(t, io.EOF, err) {
			assert.Nil(t, read)
		}
	}
}

func TestReadSSFStreamBad(t *testing.T) {
	msg := &ssf.SSFSpan{
		Version:        1,
		TraceId:        1,
		Id:             2,
		ParentId:       3,
		StartTimestamp: 9000,
		EndTimestamp:   9001,
	}

	// Bad: illegal frame:
	{
		buf := bytes.NewBuffer([]byte{0x01, 0x00})
		read, err := ReadSSF(buf, nil)
		if assert.Error(t, err) {
			assert.True(t, IsFramingError(err))
			assert.Nil(t, read)
		}
	}

	// Bad: wrong length of packet in header:
	{
		buf := bytes.NewBuffer([]byte{})
		_, err := WriteSSF(buf, msg)
		if assert.NoError(t, err) {
			// Mess with the length byte:
			buf.Bytes()[1] = 0xff
			read, err := ReadSSF(buf, nil)
			if assert.Error(t, err) {
				assert.True(t, IsFramingError(err))
			}
			assert.Nil(t, read)
		}
	}

	// Bad: invalid protobuf in SSF message:
	{
		buf := bytes.NewBuffer([]byte{})
		_, err := WriteSSF(buf, msg)
		if assert.NoError(t, err) {
			// Mess with some bytes post the length:
			buf.Bytes()[7] = 0xff
			read, err := ReadSSF(buf, nil)
			if assert.Error(t, err) {
				// This is not a framing error:
				assert.False(t, IsFramingError(err))
			}
			assert.Nil(t, read)
		}
	}
}
