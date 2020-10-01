package ftp

import (
	"bytes"
	"io/ioutil"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testData = "Just some text"
	testDir  = "mydir"
)

func TestConnPASV(t *testing.T) {
	testConn(t, true)
}

func TestConnEPSV(t *testing.T) {
	testConn(t, false)
}

func testConn(t *testing.T, disableEPSV bool) {
	assert, require := assert.New(t), require.New(t)
	mock, c := openConn(t, "127.0.0.1", DialWithTimeout(5*time.Second), DialWithDisabledEPSV(disableEPSV))

	err := c.Login("anonymous", "anonymous")
	require.NoError(err)

	err = c.NoOp()
	require.NoError(err)

	err = c.ChangeDir("incoming")
	require.NoError(err)

	dir, err := c.CurrentDir()
	require.NoError(err)
	require.Equal("/incoming", dir)

	err = c.Stor("test", bytes.NewBufferString(testData))
	require.NoError(err)

	_, err = c.List(".")
	assert.NoError(err)

	err = c.Rename("test", "tset")
	assert.NoError(err)

	// Read without deadline
	r, err := c.Retr("tset")
	assert.NoError(err)

	buf, err := ioutil.ReadAll(r)
	if assert.NoError(err) {
		assert.Equal(testData, string(buf))
	}

	r.Close()
	r.Close() // test we can close two times

	// Read with deadline
	r, err = c.Retr("tset")
	require.NoError(err)

	r.SetDeadline(time.Now())
	_, err = ioutil.ReadAll(r)
	if err == nil {
		t.Error("deadline should have caused error")
	} else if !strings.HasSuffix(err.Error(), "i/o timeout") {
		t.Error(err)
	}
	r.Close()

	// Read with offset
	r, err = c.RetrFrom("tset", 5)
	if err != nil {
		t.Error(err)
	} else {
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error(err)
		}
		expected := testData[5:]
		if string(buf) != expected {
			t.Errorf("read %q, expected %q", buf, expected)
		}
		r.Close()
	}

	data2 := bytes.NewBufferString(testData)
	err = c.Append("tset", data2)
	assert.NoError(err)

	// Read without deadline, after append
	r, err = c.Retr("tset")
	if err != nil {
		t.Error(err)
	} else {
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error(err)
		}
		if string(buf) != testData+testData {
			t.Errorf("'%s'", buf)
		}
		r.Close()
	}

	fileSize, err := c.FileSize("magic-file")
	if assert.NoError(err) {
		assert.EqualValues(42, fileSize)
	}

	_, err = c.FileSize("not-found")
	assert.EqualError(err, "550 Could not get file size.")

	assert.NoError(c.Delete("tset"))
	assert.NoError(c.MakeDir(testDir))
	assert.NoError(c.ChangeDir(testDir))
	assert.NoError(c.ChangeDirToParent())

	entries, err := c.NameList("/")
	if assert.NoError(err) {
		assert.Equal([]string{"/incoming"}, entries)
	}

	err = c.RemoveDir(testDir)
	assert.NoError(err)

	err = c.Logout()
	if err != nil {
		if protoErr := err.(*textproto.Error); protoErr != nil {
			if protoErr.Code != StatusNotImplemented {
				t.Error(err)
			}
		} else {
			t.Error(err)
		}
	}

	require.NoError(c.Quit())

	// Wait for the connection to close
	mock.Wait()

	require.Error(c.NoOp())
}

// TestConnect tests the legacy Connect function
func TestConnect(t *testing.T) {
	require := require.New(t)
	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, err := Connect(mock.Addr())
	require.NoError(err)

	require.NoError(c.Quit())

	mock.Wait()
}

func TestTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	assert := assert.New(t)

	_, err := DialTimeout("localhost:2121", 1*time.Second)
	if assert.Error(err) {
		assert.Contains(err.Error(), "connection refused")
	}
}

func TestWrongLogin(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	mock, err := newFtpMock(t, "127.0.0.1")
	require.NoError(err)
	defer mock.Close()

	c, err := DialTimeout(mock.Addr(), 5*time.Second)
	require.NoError(err)
	defer c.Quit()

	err = c.Login("zoo2Shia", "fei5Yix9")
	if assert.Error(err) {
		assert.Contains(err.Error(), "anonymous only")
	}
}

func TestDeleteDirRecur(t *testing.T) {
	require := require.New(t)

	mock, c := openConn(t, "127.0.0.1")

	err := c.RemoveDirRecur("testDir")
	require.NoError(err)

	err = c.Quit()
	require.NoError(err)

	// Wait for the connection to close
	mock.Wait()
}

// func TestFileDeleteDirRecur(t *testing.T) {
// 	mock, c := openConn(t, "127.0.0.1")

// 	err := c.RemoveDirRecur("testFile")
// 	if err == nil {
// 		t.Fatal("expected error got nil")
// 	}

// 	if err := c.Quit(); err != nil {
// 		t.Fatal(err)
// 	}

// 	// Wait for the connection to close
// 	mock.Wait()
// }

func TestMissingFolderDeleteDirRecur(t *testing.T) {
	require := require.New(t)
	mock, c := openConn(t, "127.0.0.1")

	err := c.RemoveDirRecur("missing-dir")
	require.EqualError(err, "550 missing-dir: No such file or directory")

	// Close the connection
	require.NoError(c.Quit())

	// Wait for the connection to close
	mock.Wait()
}
