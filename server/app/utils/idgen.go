package utils

import (
	"strconv"
)

type IDGenerator struct {
	counter int
	prefix  string
}

func (gen *IDGenerator) NewID() string {
	gen.counter++
	return gen.prefix + strconv.Itoa(gen.counter)
}
