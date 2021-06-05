package profile

import "fmt"

type Type int

const (
	TypeUnknown Type = iota
	TypeCPU
	TypeHeap
	TypeBlock
	TypeMutex
	TypeGoroutine
	TypeThreadCreate
)

func (t Type) String() string {
	switch t {
	case TypeUnknown:
		return "unknown"
	case TypeCPU:
		return "cpu"
	case TypeHeap:
		return "heap"
	case TypeBlock:
		return "block"
	case TypeMutex:
		return "mutex"
	case TypeGoroutine:
		return "goroutine"
	case TypeThreadCreate:
		return "threadcreate"
	default:
		return fmt.Sprintf("Type: %d", t)
	}
}
