package registry

import "errors"

var errNotHTTP = errors.New("connector: not an http name")

func errMissingFactory(name string) error {
	return errors.New("connector: missing factory " + name)
}

func errUnconstructedHTTP(name string) error {
	return errors.New("connector: unconstructed http " + name)
}
