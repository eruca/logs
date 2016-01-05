package logs

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	CRITICAL
	FATAL
)

type LogType int

func (t LogType) String() string {
	switch t {
	case LogConsole:
		return "console"
	case LogFile:
		return "file"
	}
	return ""
}

const (
	LogConsole LogType = iota
	LogFile
)

type LoggerInterface interface {
	Init()
	WriteMsg(msg string, level LogLevel) error
	Close()
	Flush()
}

// log instance generator
type LoggerGenerator func() LoggerInterface

var adapters map[LogType]LoggerGenerator

func init() {
	adapters = make(map[LogType]LoggerGenerator)

	adapters[LogConsole] = newConsole
}

type logMsg struct {
	skip  int
	level LogLevel
	msg   string
}

type Logger struct {
	sync.Mutex
	level   LogLevel
	outputs map[LogType]LoggerInterface
	msgChan chan *logMsg
	quit    chan bool
}

func NewLogger() *Logger {
	numCPU := runtime.NumCPU()

	l := newLogger(numCPU)
	l.AddLogger(LogConsole)

	return l
}

func newLogger(buffer int) *Logger {
	l := &Logger{
		level:   TRACE,
		outputs: make(map[LogType]LoggerInterface),
		msgChan: make(chan *logMsg, buffer),
		quit:    make(chan bool),
	}

	go l.startLogger()
	return l
}

func (l *Logger) AddLogger(t LogType) error {
	l.Lock()
	defer l.Unlock()

	if logGen, ok := adapters[t]; ok {
		logInst := logGen()
		if err := logInst.Init(); err != nil {
			return err
		}
		l.outputs[t] = logInst
	} else {
		panic("log: unknown adapter" + t.String())
	}
	return nil
}

func (l *Logger) DelLogger(t LogType) error {
	l.Lock()
	defer l.Unlock()

	if logInst, ok := l.outputs[t]; ok {
		logInst.Destroy()
		delete(l.outputs, t)
	} else {
		panic("log: unknown adapter" + t.String())
	}
	return nil
}

func (l *Logger) writeMsg(skip int, level LogLevel, msg string) error {
	if l.level > level {
		return nil
	}

	lm := &logMsg{
		skip:  skip,
		level: level,
	}

	if lm.level >= ERROR {
		pc, file, line, ok := runtime.Caller(skip)
		if ok {
			fn := runtime.FuncForPC(pc)
			var fnName string
			if fn == nil {
				fnName = "?()"
			} else {
				fnName = strings.TrimLeft(filepath.Ext(fn.Name()), ".") + "()"
			}

			fileName := file
			if len(fileName) > 20 {
				fileName = "..." + fileName[len(fileName)-20:]
			}
			lm.msg = fmt.Sprintf("[%s:%d %s] %s", fileName, line, fnName, msg)
		} else {
			lm.msg = msg
		}
	} else {
		lm.msg = msg
	}

	l.msgChan <- lm
	return nil
}

// 还需处理意外的情况，ctrl+c退出
func (l *Logger) startLogger() {
	for {
		select {
		case bm := <-l.msgChan:
			for _, out := range l.outputs {
				if err := l.writeMsg(bm.msg, bm.skip, bm.level); err != nil {
					fmt.Println("ERROR, unable to WriteMsg:", err)
				}
			}
		case <-l.quit:
			return
		}
	}
}

func (l *Logger) Flush() {
	for _, l := range l.outputs {
		l.Flush()
	}
}

func (l *Logger) Close() {
	l.quit <- true

}
func (l *Logger) Trace(format string, v ...interface{}) {
	msg := fmt.Sprintf("[T] "+format, v...)
	l.writerMsg(0, TRACE, msg)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	msg := fmt.Sprintf("[D] "+format, v...)
	l.writerMsg(0, DEBUG, msg)
}

func (l *Logger) Info(format string, v ...interface{}) {
	msg := fmt.Sprintf("[I] "+format, v...)
	l.writerMsg(0, INFO, msg)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	msg := fmt.Sprintf("[W] "+format, v...)
	l.writerMsg(0, WARN, msg)
}

func (l *Logger) Error(skip int, format string, v ...interface{}) {
	msg := fmt.Sprintf("[E] "+format, v...)
	l.writerMsg(skip, ERROR, msg)
}

func (l *Logger) Critical(skip int, format string, v ...interface{}) {
	msg := fmt.Sprintf("[C] "+format, v...)
	l.writerMsg(skip, CRITICAL, msg)
}

func (l *Logger) Fatal(skip int, format string, v ...interface{}) {
	msg := fmt.Sprintf("[F] "+format, v...)
	l.writerMsg(skip, FATAL, msg)
	l.Close()
	os.Exit(1)
}
