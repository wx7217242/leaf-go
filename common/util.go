package common

import (
	"time"
)

func MillisecondSinceEpoch(now time.Time) int64 {

	return now.UnixNano() / 1e6
}

// 比较两个时间毫秒值
func TimeDiffer(t1 time.Time, t2 time.Time) int64 {
	return MillisecondSinceEpoch(t1) - MillisecondSinceEpoch(t2)
}

// 返回平均时间的时间戳(毫秒）
func AverageTime(t1 time.Time, t2 time.Time) int64 {

	a := MillisecondSinceEpoch(t1)
	b := MillisecondSinceEpoch(t2)

	return b + (a-b)/2
}
