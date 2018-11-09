package common

import (
	"testing"
	"time"
	"log"
	"fmt"
)

func TestTimeDiffer(t *testing.T) {

	const timeFormatLayout = "2006-01-02 15:04:05.999999999 -0700 MST"
	t1, err := time.Parse(timeFormatLayout, "2018-10-04 20:01:19.1718396 +0800 CST")

	if err != nil {
		panic(err)
	}

	diff :=  t1.UnixNano() / 1e6
	log.Printf("%d %d", diff, t1.UnixNano())

	t2, err := time.Parse(timeFormatLayout, "2018-10-04 20:21:19.1710096 +0800 CST")
	if err != nil {
		panic(err)
	}

	msec := TimeDiffer(t2, t1)
	log.Printf("diff %d", msec)

	unix := time.Unix(0, 0)
	fmt.Println("unix:",unix.Unix())

}
