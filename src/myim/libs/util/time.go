package util

import "time"

// 秒
func GetTimestampSecond() int32 {
	return int32(time.Now().UnixNano() / (1000 * 1000 * 1000))
}

func GetTimestampMillSec() int64 {
	return int64(time.Now().UnixNano() / (1000 * 1000))
}
