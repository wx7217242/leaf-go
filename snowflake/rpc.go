package snowflake

import (
	"github.com/astaxie/beego/logs"
	"time"
)

type TimeService struct {
}

type TimeServiceReq struct {
	// 可以记录from
	// 请求时间等
	From    string
	RpcPort int
}

type TimeServiceRsp struct {
	Time string
}

const TimeFormatLayout = "2006-01-02 15:04:05.999999999 -0700 MST"

const TimeServiceMethod = "TimeService.GetSysTime"

func (t TimeService) GetSysTime(req TimeServiceReq, rsp *TimeServiceRsp) error {
	rsp.Time = time.Now().Format(TimeFormatLayout)

	logs.Info("recv rpc req from %s rpc port %d, rsp.time=%v", req.From, req.RpcPort, rsp.Time)

	return nil
}
