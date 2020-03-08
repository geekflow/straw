package internal

import "errors"

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
