package logs

import (
	"log"

	"github.com/mattn/go-colorable"
)

type console struct {
	lg *log.Logger
}

func newConsole() LoggerInterface {
	return &console{
		lg: log.New(colorable.NewColorableStdout(), "", log.Ldate|log.Ltime),
	}
}

func (c *console) WriteMsg(msg string) error {
	c.lg.Println(msg)
	return nil
}

func (*console) Flush() {}

func (*console) Destroy() {}
