package client

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// gpioSimNamePrefix is the prefix of the name the simulator gives each of its
// lines. The line at offset 7 is named "sim7", so a node can select a
// simulated line by name as well as by offset, the same as it can on a chip.
const gpioSimNamePrefix = "sim"

// gpioSimChip is the process wide state of the simulated GPIO chip. Lines are
// keyed by offset, and every line requested at one offset shares a level,
// which makes an output node and an input node at the same offset behave like
// the two ends of a wire. That is enough to exercise the whole path -- a rule
// writes valueSet on an output node and an input node sees the edge -- with no
// hardware and no root access involved.
var gpioSimChip = struct {
	mu sync.Mutex
	// levels is the level each line is at, which is what a wire carries.
	// Active low is applied by the requested line rather than stored here.
	levels map[int]bool
	// lines are the lines currently requested at each offset
	lines map[int][]*gpioSimLine
}{
	levels: map[int]bool{},
	lines:  map[int][]*gpioSimLine{},
}

// gpioSimLine is one requested line on the simulated chip
type gpioSimLine struct {
	offset    int
	activeLow bool
	// edges is nil for an output or a polled input
	edges chan<- bool
}

// gpioSimName returns the simulator's name for the line at an offset
func gpioSimName(offset int) string {
	return fmt.Sprintf("%v%v", gpioSimNamePrefix, offset)
}

// gpioSimResolve accepts either an offset or a simulated line name
func gpioSimResolve(line string) (int, error) {
	offset, err := strconv.Atoi(strings.TrimPrefix(line, gpioSimNamePrefix))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("no simulated line named %v", line)
	}

	return offset, nil
}

// gpioSimRequest requests a line on the simulated chip
func gpioSimRequest(cfg gpioLineConfig) (gpioLine, gpioLineInfo, error) {
	offset, err := gpioSimResolve(cfg.Line)
	if err != nil {
		return nil, gpioLineInfo{}, err
	}

	l := &gpioSimLine{
		offset:    offset,
		activeLow: cfg.ActiveLow,
		edges:     cfg.Edges,
	}

	gpioSimChip.mu.Lock()
	gpioSimChip.lines[offset] = append(gpioSimChip.lines[offset], l)
	gpioSimChip.mu.Unlock()

	if cfg.Output {
		if err := l.SetValue(cfg.Initial); err != nil {
			_ = l.Close()
			return nil, gpioLineInfo{}, err
		}
	}

	return l, gpioLineInfo{Offset: offset, Name: gpioSimName(offset)}, nil
}

// Value returns the active state of the line
func (l *gpioSimLine) Value() (bool, error) {
	gpioSimChip.mu.Lock()
	defer gpioSimChip.mu.Unlock()

	// an active low line reads active when the level is low
	return gpioSimChip.levels[l.offset] != l.activeLow, nil
}

// SetValue drives the line and delivers an edge to every other line requested
// at the same offset, which is what a wire between them would do. Debounce is
// not simulated, because it is a kernel behavior.
func (l *gpioSimLine) SetValue(v bool) error {
	level := v != l.activeLow

	gpioSimChip.mu.Lock()
	defer gpioSimChip.mu.Unlock()

	if gpioSimChip.levels[l.offset] == level {
		return nil
	}

	gpioSimChip.levels[l.offset] = level

	for _, other := range gpioSimChip.lines[l.offset] {
		if other == l || other.edges == nil {
			continue
		}

		// non-blocking, matching the kernel event handler, which must never
		// block the line it is delivering for
		select {
		case other.edges <- level != other.activeLow:
		default:
		}
	}

	return nil
}

// Close releases the line. The level is left as it is, the way a real line
// holds whatever the circuit around it drives.
func (l *gpioSimLine) Close() error {
	gpioSimChip.mu.Lock()
	defer gpioSimChip.mu.Unlock()

	lines := gpioSimChip.lines[l.offset]
	for i, other := range lines {
		if other == l {
			gpioSimChip.lines[l.offset] = append(lines[:i], lines[i+1:]...)
			break
		}
	}

	if len(gpioSimChip.lines[l.offset]) == 0 {
		delete(gpioSimChip.lines, l.offset)
	}

	return nil
}
