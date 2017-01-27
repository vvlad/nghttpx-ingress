package controllers

import (
	"fmt"
)

type Backend struct {
	Address  string
	Port     string
	Hostname string
	Path     string
}

type ByBackend []Backend

func (c ByBackend) Len() int      { return len(c) }
func (c ByBackend) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c ByBackend) Less(i, j int) bool {

	l := fmt.Sprintf("%s:%s", c[i].Address, c[i].Port)
	r := fmt.Sprintf("%s:%s", c[j].Address, c[j].Port)
	return l < r
}

type Hosts []string

func (c Hosts) Len() int      { return len(c) }
func (c Hosts) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c Hosts) Less(i, j int) bool {
	return c[i] < c[j]
}
