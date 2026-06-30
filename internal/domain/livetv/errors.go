package livetv

import "errors"

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrProgramNotFound = errors.New("program not found")
	ErrInvalidChannel  = errors.New("invalid channel")
	ErrInvalidProgram  = errors.New("invalid program")
)
