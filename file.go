package logs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileLogWriter struct {
	*log.Logger
	mw       *MuxWriter
	Filename string

	Maxlines          int
	maxlines_curlines int

	Maxsize         int
	maxsize_cursize int

	Daily          bool
	Maxdays        int64
	daily_opendata int

	Rotate bool

	startLock sync.Mutex

	Level LogLevel
}

type MuxWriter struct {
	sync.Mutex
	fd *os.File
}

func (l *MuxWriter) Write(b []byte) (int, error) {
	l.Lock()
	defer l.Unlock()
	return l.fd.Write(b)
}

func (l *MuxWriter) setFd(fd *os.File) {
	if l.fd != nil {
		l.fd.Close()
	}
	l.fd = fd
}

func NewFileWriter() LoggerInterface {
	w := &FileLogWriter{
		Filename: "log/log.log",
		Maxlines: 10000,
		Maxsize:  1 << 28, // 256 Mb
		Daily:    true,
		Maxdays:  7,
		Rotate:   true,
		Level:    TRACE,
	}

	w.mw = new(MuxWriter)
	w.Logger = log.New(w.mw, "", log.Ldate|log.Ltime)

	w.startLogger()

	return w
}

func (w *FileLogWriter) startLogger() error {
	fd, err := w.createLogFile()
	if err != nil {
		return err
	}

	w.mw.setFd(fd)
	if err = w.initFd(); err != nil {
		return err
	}
	return nil
}

func (w *FileLogWriter) createLogFile() (*os.File, error) {
	if !isExist(w.Filename) {
		os.MkdirAll(filepath.Dir(w.Filename), os.ModePerm)
	}
	log.Println("createLogFile")
	return os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
}

func (w *FileLogWriter) initFd() error {
	fd := w.mw.fd
	finfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("get stat: %s\n", err)
	}

	w.maxlines_curlines = int(finfo.Size())
	w.daily_opendata = time.Now().Day()
	if finfo.Size() > 0 {
		content, err := ioutil.ReadFile(w.Filename)
		if err != nil {
			return err
		}
		w.maxlines_curlines = len(strings.Split(string(content), "\n"))
	} else {
		w.maxlines_curlines = 0
	}
	return nil
}

func (w *FileLogWriter) WriteMsg(msg string, skip int, level LogLevel) error {
	if level < w.Level {
		return nil
	}
	n := 24 + len(msg) // 24 stand for the length "2013/06/23 21:00:22 [T] "
	w.docheck(n)
	w.Logger.Println(msg)
	return nil
}

func (w *FileLogWriter) docheck(size int) {
	w.startLock.Lock()
	defer w.startLock.Unlock()
	if w.Rotate && ((w.Maxlines > 0 && w.maxlines_curlines >= w.Maxlines) ||
		(w.Maxsize > 0 && w.maxsize_cursize >= w.Maxsize) ||
		(w.Daily && time.Now().Day() != w.daily_opendata)) {
		if err := w.doRotate(); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
			return
		}
	}
	w.maxlines_curlines++
	w.maxsize_cursize += size
}

func (w *FileLogWriter) doRotate() error {
	_, err := os.Lstat(w.Filename)
	if err == nil {
		num := 1
		var fname string
		for ; err == nil && num <= 999; num++ {
			fname = w.Filename + fmt.Sprintf(".%s.%03d", time.Now().Format("2006-01-02"), num)
			_, err = os.Lstat(fname)
		}

		if err == nil {
			return fmt.Errorf("rotate: cannot find free log number ot rename %s\n", w.Filename)
		}

		w.mw.Lock()
		defer w.mw.Unlock()

		fd := w.mw.fd
		fd.Close()

		if err = os.Rename(w.Filename, fname); err != nil {
			return fmt.Errorf("Rotate: %s\n", err)
		}

		if err = w.startLogger(); err != nil {
			return fmt.Errorf("Rotate StartLogger: %s\n", err)
		}

		go w.deleteOldLog()
	}

	return nil
}

func (w *FileLogWriter) deleteOldLog() {
	dir := filepath.Dir(w.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				returnErr = fmt.Errorf("Unable to delete old log '%s', error: %+v", path, r)
			}
		}()
		if !info.IsDir() && info.ModTime().Unix() < (time.Now().Unix()-60*60*24*w.Maxdays) {
			if strings.HasPrefix(filepath.Base(path), filepath.Base(w.Filename)) {
				os.Remove(path)
			}
		}
		return returnErr
	})
}

func (w *FileLogWriter) Destroy() {
	w.mw.fd.Close()
}

func (w *FileLogWriter) Flush() {
	w.mw.fd.Sync()
}
