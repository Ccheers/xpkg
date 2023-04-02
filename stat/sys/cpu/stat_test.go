package cpu

import (
	"testing"
	"time"
)

func TestStat(t *testing.T) {
	time.Sleep(time.Second * 2)
	var s Stat
	var i Info
	ReadStat(&s)
	i = GetInfo()

	t.Log("stat", s)
	t.Log("info", i)
}
