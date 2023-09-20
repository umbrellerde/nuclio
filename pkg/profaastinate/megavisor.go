package profaastinate

import (
	"runtime"
	"time"

	"github.com/nuclio/logger"
	"github.com/shirou/gopsutil/v3/cpu"
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
	boredBoundary        int
	swampedBoundary      int
	mode                 Mode
	modeChannel          chan Mode
	Logger               logger.Logger
}

func NewMegavisor(capacity int, measurementFrequency int, modeSwitchFrequency int, boredBoundary int, swampedBoundary int, logger logger.Logger) *Megavisor {
	return &Megavisor{
		make([]float64, capacity),
		capacity,
		measurementFrequency,
		modeSwitchFrequency,
		boredBoundary,
		swampedBoundary,
		Swamped, // start in swamped mode by default
		make(chan Mode),
		logger,
	}
}

func (m *Megavisor) store(value float64) {
	for i := len(m.values) - 1; i > 0; i-- {
		m.values[i] = m.values[i-1]
	}
	m.values[0] = value
}

func (m *Megavisor) avg() float64 {
	sum := 0.0
	nonNulls := len(m.values)
	for i, x := range m.values {
		if x == 0 {
			nonNulls = i
			break
		}
		sum += x
	}
	return sum / float64(nonNulls)
}

func (m *Megavisor) Start() {

	m.Logger.Debug("now in start")

	// check if Mac users want special treatment
	if runtime.GOOS == "darwin" {
		m.Logger.Error("You are not using Linux, so the Docker VM probably doesn't have access to all CPU cores. Please update numCpuFactor accordingly")
		numCpuFactor = 2
	}

	for {
		// store curr val
		usage, _ := cpu.Percent(time.Second, false)
		usageAdj := usage[0] * numCpuFactor
		m.store(usageAdj)

		// get & print avg
		avg := m.avg()
		//m.Logger.Debug("average cpu usage over last %d ms was %.2f., current is %.2f", m.modeSwitchFrequency, avg, usageAdj)

		// determine whether to switch modes
		if avg >= float64(m.swampedBoundary) && m.mode != Swamped {
			m.Logger.Info("It's my time to shine! (swamped mode activated)")
			m.mode = Swamped
			m.modeChannel <- Swamped
		}

		// else if avg <= float64(m.boredBoundary) && m.mode != Bored {
		// 	m.Logger.Info("This is boring, I could do so much more!")
		// 	m.mode = Bored
		// 	m.modeChannel <- Bored
		// }
	}
}
