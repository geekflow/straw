package process

import (
	"fmt"
	"geeksaga.com/os/straw/plugins"
	"geeksaga.com/os/straw/plugins/inputs"
	"github.com/shirou/gopsutil/process"
	"time"
)

type Process struct {
	Pid        int32   `toml:"pid"`
	Name       string  `toml:"name"`
	Exe        string  `toml:"exe"`
	Status     string  `toml:"status"`
	Uids       []int32 `toml:"uids"`
	Gids       []int32 `toml:"gids"`
	NumThreads int32   `toml:"num_threads"`
	CreateTime int64   `toml:"create_time"`
	CpuPercent float64 `toml:"cpu_percent"`
	CmdLine    string  `toml:"cmd_line"`
	NumFDs     int32   `toml:"num_fds"`
	UserName   string  `toml:"user_name"`
}

func (_ *Process) Description() string {
	return "Read metrics about process list"
}

var sampleConfig = `
`

func (_ *Process) SampleConfig() string {
	return sampleConfig
}

func (s *Process) Gather(acc plugins.Accumulator) error {
	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("error getting process list : %s", err)
	}
	now := time.Now()

	for _, p := range processes {
		exe, err := p.Exe()

		if err != nil {
			//fmt.Println(err)
			continue
		}

		name, _ := p.Name()
		status, _ := p.Status()
		uids, _ := p.Uids()
		gids, _ := p.Gids()
		numThreads, _ := p.NumThreads()
		//memoryInfo, _ := p.MemoryInfo()
		createTime, _ := p.CreateTime()
		cpuPercent, _ := p.CPUPercent()
		cmdLine, _ := p.Cmdline()
		numFDs, _ := p.NumFDs()
		userName, _ := p.Username()

		tags := map[string]string{
			"name": name,
			"exe":  exe,
		}

		fields := map[string]interface{}{
			"pid":    p.Pid,
			"status": status,
			//"parent": p.Parent(),
			//"num_ctx_switches": p.NumCtxSwitches(),
			//"sigInfo":
			"uids":        uids,
			"gids":        gids,
			"num_threads": numThreads,
			//"mem_info":    memoryInfo,
			"num_fds":    numFDs,
			"createTime": createTime,

			"cpu_percent": cpuPercent,
			"cmd_line":    cmdLine,
			//"terminal":    terminal,
			"user_name": userName,
		}

		acc.AddFields("processes", fields, tags, now)
	}

	return err
}

func init() {
	inputs.Add("process", func() plugins.Input {
		return &Process{}
	})
}
