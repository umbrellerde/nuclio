package profaastinate

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"runtime"
	"time"
)

// The Megavisor supervises the supervisors
// ...

var numCpuFactor = 1.0 // to correct cpu usage values in case docker is running in a VM on MacOS

type Mode int

const (
	Bored Mode = iota
	Swamped
)

type Megavisor struct {
	values               []float64
	capacity             int
	measurementFrequency int // in ms | recommended value:  1_000
	modeSwitchFrequency  int // in ms | recommended value: 10_000
	mode                 Mode
}

func NewMegavisor(capacity, measurementFrequency, modeSwitchFrequency int) *Megavisor {
	return &Megavisor{
		make([]float64, capacity),
		capacity,
		measurementFrequency,
		modeSwitchFrequency,
		Swamped, // start in swamped mode by default
	}
}

func (s *Megavisor) store(value float64) {
	for i := len(s.values) - 1; i > 0; i-- {
		s.values[i] = s.values[i-1]
	}
	s.values[0] = value
}

func (s *Megavisor) avg() float64 {
	sum := 0.0
	nonNulls := len(s.values)
	for i, x := range s.values {
		if x == 0 {
			nonNulls = i
			break
		}
		sum += x
	}
	return sum / float64(nonNulls)
}

func (s *Megavisor) Start() {

	// check if Mac users want special treatment
	if runtime.GOOS == "darwin" {
		fmt.Printf("You are not using Linux, so the Docker VM probably doesn't have access to all CPU cores. Please update numCpuFactor accordingly\n")
		numCpuFactor = 2
	}

	for {
		// store curr val
		usage, _ := cpu.Percent(time.Second, false)
		usageAdj := usage[0] * numCpuFactor
		s.store(usageAdj)

		// get & print avg
		avg := s.avg()
		fmt.Printf("average cpu usage over last %d ms was %.2f., current is %.2f\n", s.modeSwitchFrequency, avg, usageAdj)

		// determine whether to switch modes
		if avg > 90 && s.mode != Swamped {
			fmt.Println("now in swamped mode")
			s.mode = Swamped
		} else if avg < 80 && s.mode == Bored {
			fmt.Println("boring!")
			s.mode = Bored
		}
	}
}
