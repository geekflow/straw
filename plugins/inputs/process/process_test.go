package process

import (
	"fmt"
	"github.com/geekflow/straw/testutil"
	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessList(t *testing.T) {
	processes, err := process.Processes()

	if err != nil {
		fmt.Println(err)
	}

	assert.True(t, len(processes) > 0)

	//for _, p := range processes {
	//	exe, err := p.Exe()

	//if err != nil {
	//	continue
	//}

	//fmt.Println(exe)
	//}

	fmt.Println(process.PidExists(32039))
}

func TestGather(t *testing.T) {
	var acc testutil.Accumulator
	var err error

	err = (&Process{}).Gather(&acc)
	require.NoError(t, err)
}
