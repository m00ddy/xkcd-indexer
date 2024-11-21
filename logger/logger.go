package logger

import (
	"io"
	"log"
	"os"
	"runtime"
)

type Wood struct{
	*log.Logger
}

func InitLogger(logfile *os.File) (*Wood, error){
	// log to multiple destinations
	multi:=io.MultiWriter(logfile, os.Stdout)

	return &Wood{log.New(multi, "", log.Ltime)},nil
}

// LogInfo logs informational messages
func (w *Wood) LogInfo(message ...any) {
	w.SetPrefix(getCallerInfo() +" INFO: ")
    w.Println(message...)
}


func (w *Wood) LogDebug(message ...any) {
	w.SetPrefix(getCallerInfo() +" DEBUG: ")
    w.Println(message...)
}

// LogError logs error messages
func (w *Wood) LogError(message ...any) {
	w.SetPrefix(getCallerInfo() +" ERROR: ")
    w.Println(message...)
}

func (w *Wood) LogFatal(message ...any){
	w.SetPrefix(getCallerInfo() + "FATAL: ")
	w.Fatalln(message...)
}

func getCallerInfo()string{
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	return fn.Name()
}