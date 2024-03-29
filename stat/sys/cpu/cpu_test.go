package cpu

import (
	"fmt"
	"testing"
	"time"
)

func Test_CPUUsage(t *testing.T) {
	var stat Stat
	ReadStat(&stat)
	fmt.Println(stat)
	time.Sleep(time.Millisecond * 1000)
	for i := 0; i < 6; i++ {
		time.Sleep(time.Millisecond * 500)
		ReadStat(&stat)
		t.Log("stat", stat)
	}
}
