package logger

import (
    "io"
    "log"
    "strings"
    "github.com/fatih/color"
)

type Logger struct{
    Trace   *log.Logger
    Info    *log.Logger
    Warning *log.Logger
    Error   *log.Logger
    Panic   *log.Logger
}

func NewLogger(
    name string,
    traceHandle io.Writer, 
    infoHandle io.Writer, 
    warningHandle io.Writer, 
    errorHandle io.Writer, 
    panicHandle io.Writer) *Logger{
    l := new(Logger)

    if strings.Contains(name, "SERVER") {
        name = color.BlueString(name)
    }else if strings.Contains(name, "CLIENT") {
        name = color.CyanString(name)
    }

    l.Trace = log.New(traceHandle,
        name + "[  TRACE  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Info = log.New(infoHandle,
        name + "[  INFO   ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Warning = log.New(warningHandle,
        name + color.RedString("[ WARNING ]: "),
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Error = log.New(errorHandle,
        name + color.RedString("[  ERROR  ]: "),
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Panic = log.New(errorHandle,
        name + color.RedString("[  PANIC  ]: "),
        log.Ldate|log.Ltime|log.Lshortfile)

    return l
}
