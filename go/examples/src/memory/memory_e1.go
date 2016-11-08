package main

import (
	"fmt"
	"github.com/shirou/gopsutil/process"
	"os"
)

var ps *process.Process

func mem(n int) {
	if ps == nil {
		p, err := process.NewProcess(int32(os.Getpid()))
		if err != nil {
			panic(err)
		}
		ps = p
	}
	mem, _ := ps.MemoryInfo()
	fmt.Printf("%d. VMS: %dGB, RSS: %dMB, Swap: %dMB\n", n, mem.VMS>>30, mem.RSS>>20, mem.Swap>>20)
}

func main() {
	mem(1)
	// 10M memory
	data := new([10][1024 * 1024]byte)
	for i := range data {
		for x, n := 0, len(data[i]); x < n; x++ {
			data[i][x] = 1
		}
		mem(2)
	}
	mem(3)
}
