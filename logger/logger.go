package logger

import (
    "io"
    "log"
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

    l.Trace = log.New(traceHandle,
        name + "[  TRACE  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Info = log.New(infoHandle,
        name + "[  INFO   ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Warning = log.New(warningHandle,
        name + "[ WARNING ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Error = log.New(errorHandle,
        name + "[  ERROR  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Panic = log.New(errorHandle,
        name + "[  PANIC  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    return l
}
