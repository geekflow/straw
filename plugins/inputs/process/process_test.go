package process

import (
	"fmt"
	"github.com/shirou/gopsutil/process"
	"testing"
)

func TestProcessList(t *testing.T) {
	processes, err := process.Processes()

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(len(processes))

	for _, p := range processes {
		name, err := p.Exe()

		if err != nil {
			continue
		}

		fmt.Println(name)
	}

	fmt.Println(process.PidExists(32039))
}
