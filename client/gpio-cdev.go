//go:build linux

package client

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/simpleiot/simpleiot/data"
	"github.com/warthog618/go-gpiocdev"
)

// gpioConsumer is the name this client registers with the kernel, which is
// what gpioinfo shows against a line Simple IoT holds
const gpioConsumer = "simpleiot"

// gpioCdevLine adapts a requested character device line to the gpioLine
// interface, which works in booleans rather than the kernel's 0 and 1.
type gpioCdevLine struct {
	line *gpiocdev.Line
}

func (l *gpioCdevLine) Value() (bool, error) {
	v, err := l.line.Value()
	if err != nil {
		return false, err
	}

	return v != 0, nil
}

func (l *gpioCdevLine) SetValue(v bool) error {
	if v {
		return l.line.SetValue(1)
	}

	return l.line.SetValue(0)
}

func (l *gpioCdevLine) Close() error {
	return l.line.Close()
}

// gpioCdevChip opens the named chip. The name is a chip name such as
// "gpiochip0" or a full device path; when neither matches, the chips on the
// system are searched for one with that label, which is how a board describes
// an expander that does not always land on the same chip number.
func gpioCdevChip(name string) (*gpiocdev.Chip, error) {
	chip, err := gpiocdev.NewChip(name, gpiocdev.WithConsumer(gpioConsumer))
	if err == nil {
		return chip, nil
	}

	for _, n := range gpiocdev.Chips() {
		c, cErr := gpiocdev.NewChip(n, gpiocdev.WithConsumer(gpioConsumer))
		if cErr != nil {
			continue
		}

		if c.Label == name {
			return c, nil
		}

		_ = c.Close()
	}

	return nil, fmt.Errorf("error opening chip %v: %w", name, err)
}

// gpioCdevResolve accepts either a line offset or the kernel's name for the
// line, so a node can be written against a board's line names rather than
// against offsets that move between kernel versions.
func gpioCdevResolve(chip *gpiocdev.Chip, line string) (int, error) {
	if offset, err := strconv.Atoi(line); err == nil {
		if offset < 0 || offset >= chip.Lines() {
			return 0, fmt.Errorf("line %v is out of range for %v, which has %v lines",
				offset, chip.Name, chip.Lines())
		}

		return offset, nil
	}

	offset, err := chip.FindLine(line)
	if err != nil {
		return 0, fmt.Errorf("no line named %v on %v", line, chip.Name)
	}

	return offset, nil
}

// gpioCdevRequestError explains a failed request. A line already claimed by a
// driver is the common case, and the kernel knows which driver holds it, which
// is the question a chip listing would otherwise be used to answer.
func gpioCdevRequestError(chip *gpiocdev.Chip, offset int, err error) error {
	if errors.Is(err, gpiocdev.ErrPermissionDenied) {
		return fmt.Errorf("permission denied requesting line %v on %v: %w",
			offset, chip.Name, err)
	}

	info, infoErr := chip.LineInfo(offset)
	if infoErr == nil && info.Used && info.Consumer != "" {
		return fmt.Errorf("line %v on %v is held by %v: %w",
			offset, chip.Name, info.Consumer, err)
	}

	return fmt.Errorf("error requesting line %v on %v: %w", offset, chip.Name, err)
}

// gpioCdevRequest requests one line on a GPIO character device. Closing the
// chip does not disturb a line requested from it, so the chip is released as
// soon as the line is in hand.
func gpioCdevRequest(cfg gpioLineConfig) (gpioLine, gpioLineInfo, error) {
	chip, err := gpioCdevChip(cfg.Chip)
	if err != nil {
		return nil, gpioLineInfo{}, err
	}

	defer func() {
		_ = chip.Close()
	}()

	offset, err := gpioCdevResolve(chip, cfg.Line)
	if err != nil {
		return nil, gpioLineInfo{}, err
	}

	opts := []gpiocdev.LineReqOption{gpiocdev.WithConsumer(gpioConsumer)}

	if cfg.Output {
		v := 0
		if cfg.Initial {
			v = 1
		}
		opts = append(opts, gpiocdev.AsOutput(v))

		switch cfg.Drive {
		case data.PointValueOpenDrain:
			opts = append(opts, gpiocdev.AsOpenDrain)
		case data.PointValueOpenSource:
			opts = append(opts, gpiocdev.AsOpenSource)
		}
	} else {
		opts = append(opts, gpiocdev.AsInput)

		if cfg.Debounce > 0 {
			opts = append(opts, gpiocdev.WithDebounce(cfg.Debounce))
		}

		if cfg.Edges != nil {
			edges := cfg.Edges
			opts = append(opts, gpiocdev.WithBothEdges,
				gpiocdev.WithEventHandler(func(e gpiocdev.LineEvent) {
					// the handler runs on the event reading goroutine and
					// must not block, so an event is dropped rather than
					// stalling every later event on the line
					select {
					case edges <- e.Type == gpiocdev.LineEventRisingEdge:
					default:
					}
				}))
		}
	}

	switch cfg.Bias {
	case data.PointValuePullUp:
		opts = append(opts, gpiocdev.WithPullUp)
	case data.PointValuePullDown:
		opts = append(opts, gpiocdev.WithPullDown)
	case data.PointValueBiasDisabled:
		opts = append(opts, gpiocdev.WithBiasDisabled)
	}

	if cfg.ActiveLow {
		opts = append(opts, gpiocdev.AsActiveLow)
	}

	info := gpioLineInfo{Offset: offset}
	if li, liErr := chip.LineInfo(offset); liErr == nil {
		info.Name = li.Name
	}

	line, err := chip.RequestLine(offset, opts...)
	if err != nil {
		return nil, gpioLineInfo{}, gpioCdevRequestError(chip, offset, err)
	}

	return &gpioCdevLine{line: line}, info, nil
}
