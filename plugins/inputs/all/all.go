package all

import (
	_ "github.com/geekflow/straw/plugins/inputs/cpu"
	_ "github.com/geekflow/straw/plugins/inputs/disk"
	_ "github.com/geekflow/straw/plugins/inputs/mem"
	_ "github.com/geekflow/straw/plugins/inputs/net"
	_ "github.com/geekflow/straw/plugins/inputs/process"
	_ "github.com/geekflow/straw/plugins/inputs/procstat"
)
