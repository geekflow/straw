package outputs

import (
	"github.com/geekflow/straw/plugins"
)

type Creator func() plugins.Output

var Outputs = map[string]Creator{}

func Add(name string, creator Creator) {
	Outputs[name] = creator
}
