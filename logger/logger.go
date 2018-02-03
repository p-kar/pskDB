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
    traceHandle io.Writer, 
    infoHandle io.Writer, 
    warningHandle io.Writer, 
    errorHandle io.Writer, 
    panicHandle io.Writer) *Logger{
    l := new(Logger)

    l.Trace = log.New(traceHandle,
        "[  TRACE  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Info = log.New(infoHandle,
        "[  INFO   ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Warning = log.New(warningHandle,
        "[ WARNING ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Error = log.New(errorHandle,
        "[  ERROR  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    l.Panic = log.New(errorHandle,
        "[  PANIC  ]: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    return l
}
