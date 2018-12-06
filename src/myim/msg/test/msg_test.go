package test

import (
	"myim/libs/define"
	proto "myim/libs/proto"
	"myim/libs/util"
	rpc "net/rpc"
	"strconv"
	"testing"
)

func TestSaveMsg(t *testing.T) {
	c, err := rpc.Dial("tcp", "localhost:7270")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	for i := 0; i < 40; i++ {
		args := proto.SaveMsgArg{
			AppId:      "bilin",
			ChatType:   1,
			ChatCode:   "123_456",
			FromUserId: "123",
			MsgData:    "hello" + strconv.Itoa(i),
			Tag:        "mytag",
			CreateTime: 12345,
		}
		reply := proto.SaveMsgReply{}
		if err = c.Call("MsgRPC.SaveMsg", &args, &reply); err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Logf("save msg reply:%v", reply)
	}
}

func TestGetMsgList(t *testing.T) {
	c, err := rpc.Dial("tcp", "localhost:7270")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	args := proto.GetMsgListArg{
		AppId:     "bilin",
		ChatType:  1,
		ChatCode:  "123_456",
		Direction: define.DIRECTION_BACKWARD,
		Count:     21,
	}
	reply := proto.GetMsgListReply{}
	t1 := util.GetTimestampMillSec()
	for i := 0; i < 1000; i++ {
		if err = c.Call("MsgRPC.GetMsgList", &args, &reply); err != nil {
			t.Error(err)
			t.FailNow()
		}
	}
	t2 := util.GetTimestampMillSec()
	t.Logf("get msg reply:%v, ms:%d", reply, t2-t1)
}
