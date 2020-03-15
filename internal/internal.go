package internal

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/units"
)

var (
	VersionAlreadySetError = errors.New("version has already been set")
)

var version string

func SetVersion(v string) error {
	if version != "" {
		return VersionAlreadySetError
	}
	version = v
	return nil
}

func Version() string {
	return version
}

// Duration just wraps time.Duration
type Duration struct {
	Duration time.Duration
}

// Size just wraps an int64
type Size struct {
	Size int64
}

type Number struct {
	Value float64
}

// ProductToken returns a tag for Flow that can be used in user agents.
func ProductToken() string {
	return fmt.Sprintf("Flow/%s Go/%s",
		Version(), strings.TrimPrefix(runtime.Version(), "go"))
}

// UnmarshalTOML parses the duration from the TOML config file
func (d *Duration) UnmarshalTOML(b []byte) error {
	var err error
	b = bytes.Trim(b, `'`)

	// see if we can directly convert it
	d.Duration, err = time.ParseDuration(string(b))
	if err == nil {
		return nil
	}

	// Parse string duration, ie, "1s"
	if uq, err := strconv.Unquote(string(b)); err == nil && len(uq) > 0 {
		d.Duration, err = time.ParseDuration(uq)
		if err == nil {
			return nil
		}
	}

	// First try parsing as integer seconds
	sI, err := strconv.ParseInt(string(b), 10, 64)
	if err == nil {
		d.Duration = time.Second * time.Duration(sI)
		return nil
	}
	// Second try parsing as float seconds
	sF, err := strconv.ParseFloat(string(b), 64)
	if err == nil {
		d.Duration = time.Second * time.Duration(sF)
		return nil
	}

	return nil
}

func (s *Size) UnmarshalTOML(b []byte) error {
	var err error
	b = bytes.Trim(b, `'`)

	val, err := strconv.ParseInt(string(b), 10, 64)
	if err == nil {
		s.Size = val
		return nil
	}
	uq, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	val, err = units.ParseStrictBytes(uq)
	if err != nil {
		return err
	}
	s.Size = val
	return nil
}

func (n *Number) UnmarshalTOML(b []byte) error {
	value, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}

	n.Value = value
	return nil
}

// RandomDuration returns a random duration between 0 and max.
func RandomDuration(max time.Duration) time.Duration {
	if max == 0 {
		return 0
	}

	var sleepns int64
	maxSleep := big.NewInt(max.Nanoseconds())
	if j, err := rand.Int(rand.Reader, maxSleep); err == nil {
		sleepns = j.Int64()
	}

	return time.Duration(sleepns)
}

// SleepContext sleeps until the context is closed or the duration is reached.
func SleepContext(ctx context.Context, duration time.Duration) error {
	if duration == 0 {
		return nil
	}

	t := time.NewTimer(duration)
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		t.Stop()
		return ctx.Err()
	}
}

// AlignDuration returns the duration until next aligned interval.
// If the current time is aligned a 0 duration is returned.
func AlignDuration(tm time.Time, interval time.Duration) time.Duration {
	return AlignTime(tm, interval).Sub(tm)
}

// AlignTime returns the time of the next aligned interval.
// If the current time is aligned the current time is returned.
func AlignTime(tm time.Time, interval time.Duration) time.Time {
	truncated := tm.Truncate(interval)
	if truncated == tm {
		return tm
	}
	return truncated.Add(interval)
}

// CompressWithGzip takes an io.Reader as input and pipes
// it through a gzip.Writer returning an io.Reader containing
// the gzipped data.
// An error is returned if passing data to the gzip.Writer fails
func CompressWithGzip(data io.Reader) (io.Reader, error) {
	pipeReader, pipeWriter := io.Pipe()
	gzipWriter := gzip.NewWriter(pipeWriter)

	var err error
	go func() {
		_, err = io.Copy(gzipWriter, data)
		gzipWriter.Close()
		// subsequent reads from the read half of the pipe will
		// return no bytes and the error err, or EOF if err is nil.
		pipeWriter.CloseWithError(err)
	}()

	return pipeReader, err
}
