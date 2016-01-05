package logs

import (
	"log"

	"github.com/mattn/go-colorable"
)

type Brush func(string) string

func NewBrush(color string) Brush {
	pre := "\033["
	reset := "\033[0m"
	return func(text string) string {
		return pre + color + "m" + text + reset
	}
}

var colors = []Brush{
	NewBrush("1;36"), // Trace      cyan
	NewBrush("1;34"), // Debug      blue
	NewBrush("1;32"), // Info       green
	NewBrush("1;33"), // Warn       yellow
	NewBrush("1;31"), // Error      red
	NewBrush("1;35"), // Critical   purple
	NewBrush("1;31"), // Fatal      red
}

type console struct {
	level LogLevel
	lg    *log.Logger
}

func newConsole() LoggerInterface {
	return &console{
		lg: log.New(colorable.NewColorableStdout(), "", log.Ldate|log.Ltime),
	}
}

func (c *console) WriteMsg(msg string, skip int, level LogLevel) error {
	log.Println(msg, skip, level)
	if c.level > level {
		return nil
	}

	c.lg.Println(colors[level](msg))
	return nil
}

func (*console) Flush() {}

func (*console) Destroy() {}
