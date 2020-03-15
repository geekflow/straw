package inputs

import "geeksaga.com/os/straw/plugins"

type Creator func() plugins.Input

var Inputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Inputs[name] = creator
}
