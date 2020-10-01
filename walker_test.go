package ftp

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkReturnsCorrectlyPopulatedWalker(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.NoError(cErr)

	w := c.Walk("root")
	assert.Equal("root/", w.root)
	assert.Equal(&c, &w.serverConn)
}

func TestFieldsReturnCorrectData(t *testing.T) {
	assert := assert.New(t)

	w := Walker{
		cur: &item{
			path: "/root/",
			err:  errors.New("this is an error"),
			entry: &Entry{
				Name: "root",
				Size: 123,
				Time: time.Now(),
				Type: EntryTypeFolder,
			},
		},
	}

	assert.Equal("this is an error", w.Err().Error())
	assert.Equal("/root/", w.Path())
	assert.Equal(EntryTypeFolder, w.Stat().Type)
}

func TestSkipDirIsCorrectlySet(t *testing.T) {
	w := Walker{}
	w.SkipDir()

	assert.Equal(t, false, w.descend)
}

func TestNoDescendDoesNotAddToStack(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.NoError(cErr)

	w := c.Walk("/root")
	w.cur = &item{
		path: "/root/",
		err:  nil,
		entry: &Entry{
			Name: "root",
			Size: 123,
			Time: time.Now(),
			Type: EntryTypeFolder,
		},
	}

	w.stack = []*item{
		{
			path: "file",
			err:  nil,
			entry: &Entry{
				Name: "file",
				Size: 123,
				Time: time.Now(),
				Type: EntryTypeFile,
			},
		},
	}

	w.SkipDir()
	assert.True(w.Next())
	assert.Empty(w.stack)
	assert.True(w.descend)
}

func TestEmptyStackReturnsFalse(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.NoError(cErr)

	w := c.Walk("/root")

	w.cur = &item{
		path: "/root/",
		err:  nil,
		entry: &Entry{
			Name: "root",
			Size: 123,
			Time: time.Now(),
			Type: EntryTypeFolder,
		},
	}

	w.stack = []*item{}
	w.SkipDir()

	assert.False(w.Next())
}

func TestCurAndStackSetCorrectly(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.NoError(cErr)

	w := c.Walk("/root")
	w.cur = &item{
		path: "root/file1",
		err:  nil,
		entry: &Entry{
			Name: "file1",
			Size: 123,
			Time: time.Now(),
			Type: EntryTypeFile,
		},
	}

	w.stack = []*item{
		{
			path: "file",
			err:  nil,
			entry: &Entry{
				Name: "file",
				Size: 123,
				Time: time.Now(),
				Type: EntryTypeFile,
			},
		},
		{
			path: "root/file1",
			err:  nil,
			entry: &Entry{
				Name: "file1",
				Size: 123,
				Time: time.Now(),
				Type: EntryTypeFile,
			},
		},
	}

	assert.True(w.Next())
	assert.True(w.Next())
	assert.Empty(w.stack)
	assert.Equal("file", w.cur.entry.Name)
}

func TestCurInit(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, cErr := Connect(mock.Addr())
	require.NoError(cErr)

	w := c.Walk("/root")

	assert.True(w.Next())
	// mock fs has one file 'lo'

	assert.Empty(w.stack)
	assert.Equal("/root/lo", w.Path())
}
