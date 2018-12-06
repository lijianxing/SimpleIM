package main

import (
	"fmt"
	"strconv"
	"strings"
)

func encodeUserKey(appId string, userId string, seq int32) string {
	return fmt.Sprintf("%s_%s_%d", appId, userId, seq)
}

func decodeUserKey(key string) (appId string, userId string, seq int32, err error) {
	var (
		idx  int
		idx2 int
		t    int64
	)

	if idx = strings.IndexByte(key, '_'); idx == -1 {
		err = ErrDecodeKey
		return
	} else {
		appId = key[:idx]

		remain := key[idx+1:]
		if idx2 = strings.LastIndexByte(remain, '_'); idx2 == -1 {
			err = ErrDecodeKey
			return
		} else {
			userId = remain[:idx2]
			if t, err = strconv.ParseInt(remain[idx2+1:], 10, 32); err != nil {
				return
			}
			seq = int32(t)
		}
	}
	return
}

func encodeRouteKey(appId string, userId string) string {
	return fmt.Sprintf("%s_%s", appId, userId)
}

func encodeSinleSessionCode(userId1, userId2 string) string {
	var first string
	var second string
	if userId1 < userId2 {
		first = userId1
		second = userId2
	} else {
		first = userId2
		second = userId1
	}
	return fmt.Sprintf("%s_%s", first, second)
}

func encodeGroupSessionCode(groupCode string) string {
	return groupCode
}
