package log

import (
	"io"

	"gopkg.in/natefinch/lumberjack.v2"
)

type WriterConfig struct {
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxsize"`
	MaxAge     int    `json:"maxage"`
	MaxBackups int    `json:"maxbackups"`
	LocalTime  bool   `json:"localtime"`
	Compress   bool   `json:"compress"`
}

func NewWriter(cfg *WriterConfig) (io.Writer, error) {
	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}, nil
}
