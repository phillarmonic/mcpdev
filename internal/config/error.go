package config

import (
	"fmt"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

func configErr(path, field, msg string) error {
	switch {
	case path != "" && field != "":
		return errclass.Usage{Msg: fmt.Sprintf("config: %s: %s: %s", path, field, msg)}
	case path != "":
		return errclass.Usage{Msg: fmt.Sprintf("config: %s: %s", path, msg)}
	case field != "":
		return errclass.Usage{Msg: fmt.Sprintf("config: %s: %s", field, msg)}
	default:
		return errclass.Usage{Msg: "config: " + msg}
	}
}

func profileField(name, field string) string {
	if field == "" {
		return "profiles." + name
	}
	return "profiles." + name + "." + field
}
