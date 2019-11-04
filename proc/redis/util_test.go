package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCommandFromResp(t *testing.T) {
	for _, c := range []struct {
		Input   *RespValue
		Command string
	}{
		{
			Input:   newStringArray("get", "a"),
			Command: "get",
		},
		{
			Input:   newStringArray("GET", "a"),
			Command: "get",
		},
		{
			Input:   newStringArray("set", "a", "b"),
			Command: "set",
		},
		{
			Input:   newStringArray("SET", "a", "b"),
			Command: "set",
		},
		{
			Input:   nil,
			Command: "",
		},
		{
			Input:   newStringArray(),
			Command: "",
		},
		{
			Input:   newStringArray(""),
			Command: "",
		},
	} {
		assert.Equal(t, c.Command, getCommandFromResp(c.Input))
	}
}
