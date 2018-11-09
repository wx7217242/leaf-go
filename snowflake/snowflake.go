package snowflake

import (
	"common"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// 上报到leaf forever node的数据，Host 表明是具体的哪一台主机，HttpPort表明是提供id服务的端口，time是定时上传的时间
type LeafForeverNodeData struct {
	Host     string `json:"Host"`
	HttpPort int    `json:"HttpPort"`
	Time     string `json:"Time"`
}

type LeafSnowFlakeIdResp struct {
	Id  int64  `json:"id"`
	Msg string `json:"msg"`
}

const (
	IdReqPrefix       = "/snowflake/id"
	leafForeverPrefix = "/leaf-forever"
	leafTempPrefix    = "/leaf-temp"
	allowedOffset     = 5 // 毫秒
	sequenceMax       = 1 << 11

	// 生成id的时期
	genIdEpoch = "2018-10-01 00:00:00 +0800 CST"
)

func workIdToInt(workId string) (int, error) {
	trim := strings.Trim(workId, leafForeverPrefix)
	return strconv.Atoi(trim)
}

type SnowFlake struct {
	intLeafForeverWorkId int
	leafTempNodeWorkId   string
	sequence             int64
	zk                   ZKLeafClient
	ticker               *time.Ticker
	lastGenIdTimestamp   int64 // 上一次生成的时间
	genIdEpochMSec       int64 // 生成id的时候，当前时间的时间戳减去这个值
}

func (s *SnowFlake) ServeIdReq(w http.ResponseWriter, req *http.Request) {

	id := s.GenerateId()

	// logs.Debug("recv req from %s, id %d", req.RemoteAddr, id)

	res := LeafSnowFlakeIdResp{}
	res.Id = id

	// 返回json格式的数据，
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	// HTTP 1.1默认进行持久连接
	w.Header().Set("Connection", "close")

	if id == -1 {
		w.WriteHeader(http.StatusInternalServerError)
		res.Msg = "server internal error"
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		logs.Error("json encode %v failed", res)
	}
}

func (s *SnowFlake) getCurrentMillisecond() (int64, bool) {

	now := time.Now()

	// 取当前时间的毫秒值
	curMSec := common.MillisecondSinceEpoch(now)

	lastMSec := atomic.LoadInt64(&s.lastGenIdTimestamp)

	// 没有回拨
	if lastMSec <= curMSec {
		return curMSec, false
	}

	// 发生了回拨
	offset := lastMSec - curMSec

	// 发生了回拨，比允许的偏差大
	if offset > allowedOffset {
		logs.Warning("clock backwards, offset %d > allowed %d", offset, allowedOffset)
		return curMSec, true
	}

	// 发生了回拨，比允许的偏差小，等待两倍时间
	after := time.After(time.Duration(time.Millisecond * allowedOffset * 2))
	select {
	case <-after:
		curMSec = common.MillisecondSinceEpoch(time.Now()) // 当前时间的毫秒数
		lastMSec = atomic.LoadInt64(&s.lastGenIdTimestamp) // 上一次发号的毫秒
	}

	if curMSec < lastMSec {
		logs.Warning("after wait %d ms, clock still backward", allowedOffset*2)
		return curMSec, true
	} else {
		logs.Info("after wait %d ms, clock catch up", allowedOffset*2)
		return curMSec, false
	}

}

func (s *SnowFlake) GenerateId() int64 {

	if s.sequence > sequenceMax {
		s.sequence = 0
	}

	curMSec, back := s.getCurrentMillisecond()
	if back {
		return -1
	}

	atomic.StoreInt64(&s.lastGenIdTimestamp, curMSec)

	// 当前时间减去开始时间
	curMSec = curMSec - s.genIdEpochMSec

	// 1+41+10+12
	id := curMSec<<22 | int64(s.intLeafForeverWorkId<<12) | s.sequence
	s.sequence++

	return id
}

func (s *SnowFlake) Start(config *common.Config) error {

	begin, err := time.Parse(TimeFormatLayout, genIdEpoch)
	if err != nil {
		logs.Error("parse %s failed err %v", genIdEpoch, err)
		return err
	}
	s.genIdEpochMSec = common.MillisecondSinceEpoch(begin)

	err = s.checkSysTime(config)
	if err != nil {
		logs.Error("check systime failed err %v", err)
		return err
	}

	// 创建rpc服务
	rpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.RpcServicePort))
	if err != nil {
		return err
	}

	rpc.Register(TimeService{})

	go func() {
		for {
			conn, err := rpcListener.Accept()
			if err != nil {
				logs.Error("rpc listener accept client failed, err %v", err)
				continue
			}

			logs.Debug("accept rpc client from %v", conn.RemoteAddr())

			go jsonrpc.ServeConn(conn)
		}
	}()

	return nil
}

func (s *SnowFlake) Run(config common.Config) {

	defer logs.Info("Run over...")

	// 创建对外的http服务
	addr := fmt.Sprintf(":%d", config.HttpPort)

	// 运行在一个http服务里面
	in := make(chan error)
	go func() {
		http.HandleFunc(IdReqPrefix, s.ServeIdReq)

		err := http.ListenAndServe(addr, nil)
		in <- err
	}()

	logs.Info("leaf service serve on %s", addr)

	var rpcHost = fmt.Sprintf("%s:%d", config.LeafHost, config.RpcServicePort)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	for {
		select {
		case err := <-in:
			logs.Info("leaf service recv error %v", err)
			// 这里使用break不能跳出for循环，
			return
		case sig := <-c:
			logs.Info("leaf service recv signal %v", sig)
			return
		case <-s.ticker.C:
			err := s.reportSysTime(config)
			if err != nil {
				logs.Error("report systime failed, err %v", err)
			}
		case event := <-s.zk.Event:
			logs.Debug("zkclient event state:%v type:%v path:%v err:%v server:%v zkclient addr: %p",
				event.State, event.Type, event.Path, event.Err, event.Server, &s.zk)

			// TODO: 有点小BUG，zk掉线重连，会收到多个event
			if event.State != zk.StateConnected {
				continue
			}

			logs.Info("delete leaf temp node %v first", rpcHost)
			s.zk.DeleteLeafTempNode(rpcHost)

			if leafTempNodeWorkId, err := s.zk.CreateLeafTempNode(rpcHost); err != nil {
				logs.Error("create temp node %s failed, err %v", rpcHost, err)
			} else {
				logs.Info("create temp node %s succeed", leafTempNodeWorkId)
				s.leafTempNodeWorkId = leafTempNodeWorkId

				s.reportSysTime(config)
			}

		}
	}

}

func (s *SnowFlake) checkSysTime(config *common.Config) error {

	// config.ZookeeperHost可以配置多个，以,分割
	zkHosts := strings.Split(config.ZookeeperHost, ",")

	err := s.zk.Connect(zkHosts)
	if err != nil {
		return err
	}

	if behind, err := s.isLocalTimeLeftBehind(config.LeafHost, config.RpcServicePort, config.TimeThresholdMS); err != nil {
		return err
	} else {
		if behind {
			return errors.New("local time left behind, please check")
		}
	}

	leafForeverWorkId := config.LeafForeverWorkId
	if leafForeverWorkId == "" {
		//  创建持久节点
		workId, err := s.zk.CreateLeafForeverNode()
		if err != nil {
			logs.Error("create forever node failed! err %v", err)
			return err
		} else {
			logs.Info("create forever node succeed! workid %s", workId)
			s.intLeafForeverWorkId, _ = workIdToInt(workId)
			config.LeafForeverWorkId = workId
			config.SaveLeafForeverWorkId() // 把持久顺序节点的id保存到配置文件中
		}
	} else {
		// 获取持久节点之前保存的数据，判断时间是否落后

		nodeData, err := s.zk.GetLeafForeverNodeData(leafForeverWorkId)
		if err != nil {
			logs.Error("get forever node data failed! err %v, workId %s not exist", err, leafForeverWorkId)
			return err
		}

		// 获取上传到持久顺序节点的信息
		var data LeafForeverNodeData
		err = json.Unmarshal([]byte(nodeData), &data)
		if err != nil {
			logs.Error("json unmarshal forever node data %v failed", nodeData)
			return err
		}

		if data.Host != "" && data.Host != config.LeafHost {
			return fmt.Errorf("forever node host [%v] not equal to config's host [%v]",
				data.Host, config.LeafHost)
		}

		if data.HttpPort != 0 && data.HttpPort != config.HttpPort {
			return fmt.Errorf("forever node http port [%d] not equal to config's http port [%d]",
				data.HttpPort, config.HttpPort)
		}

		if data.Time == "" {
			return fmt.Errorf("forever node data.time `%v` is empty", data.Time)
		}

		lastSavedTime, err := time.Parse(TimeFormatLayout, data.Time)
		if err != nil {
			logs.Error("parse time %v failed", data.Time)
			return err
		}

		now := time.Now()
		if lastSavedTime.After(now) {
			return fmt.Errorf("lastSavedTime %v is after now %v", lastSavedTime, now)
		}
	}

	// 创建定时器，周期性上传时间
	ticker := time.NewTicker(3 * time.Second)
	s.ticker = ticker

	return nil
}

func (s *SnowFlake) reportSysTime(config common.Config) error {
	var data LeafForeverNodeData
	data.Host = config.LeafHost
	data.HttpPort = config.HttpPort
	data.Time = time.Now().Format(TimeFormatLayout)
	bytes, err := json.Marshal(data)
	if err != nil {
		logs.Error("json marshal %v faild", data)
		return err
	}

	err = s.zk.SetLeafForeverNodeData(config.LeafForeverWorkId, string(bytes))
	if err != nil {
		logs.Error("set data %v to %s faild, err %v", string(bytes), config.LeafForeverWorkId, err)
		return err
	}

	return nil
}

// 校验时间
func (s *SnowFlake) isLocalTimeLeftBehind(leafHost string, rpcPort int, timeThresholdMS int) (bool, error) {

	children, err := s.zk.GetLeafTempChildren()
	if err != nil {
		logs.Error("get leaf temp children faild err %v", err)
		return false, err
	}

	logs.Info("leaf temp children %v", children)
	// 排除自己
	localRpcHost := fmt.Sprintf("%s:%d", leafHost, rpcPort)
	for _, c := range children {
		if c == localRpcHost {
			err = fmt.Errorf("local %s is running, make sure only one instance", localRpcHost)
			logs.Error("%v", err.Error())
			return false, err
		}
	}

	// rpc 获取 临时节点的时间，然后跟本机时间做对比，判断是否落后
	if len(children) == 0 {
		logs.Info("there is no other leaf-temp node")
		return false, nil
	}

	for _, c := range children {

		remoteRpcHost := c
		logs.Debug("create json rpc call to %s", remoteRpcHost)

		start := time.Now()

		client, err := jsonrpc.Dial("tcp", remoteRpcHost)
		if err != nil {
			return false, err
		}

		logs.Debug("json rpc dial to %s succeed", remoteRpcHost)

		req := TimeServiceReq{From: leafHost, RpcPort: rpcPort}
		rsp := TimeServiceRsp{}
		err = client.Call(TimeServiceMethod, req, &rsp)
		if err != nil {
			return false, err
		}
		client.Close() // important
		end := time.Now()

		avg := common.AverageTime(start, end)

		logs.Debug("rpc call return, req %v, rsp %v", req, rsp)
		rpcTime, err := time.Parse(TimeFormatLayout, rsp.Time)
		if err != nil {
			return false, err
		}

		diff := math.Abs(float64(common.MillisecondSinceEpoch(rpcTime) - avg))
		if diff <= float64(timeThresholdMS) {
			logs.Warning("local %s:%d with %s time diff %f'ms is not in 0-%dms",
				leafHost, rpcPort, remoteRpcHost, diff, timeThresholdMS)
		} else {
			logs.Info("local %s:%d with %s time diff %f'ms is in 0-%dms",
				leafHost, rpcPort, remoteRpcHost, diff, timeThresholdMS)
			return true, nil
		}

	}
	return false, nil
}

func (s *SnowFlake) Stop() {
	// 停止定时器
	s.ticker.Stop()

	// 停止http服务和rpc服务

	// 删除临时节点
	s.zk.DeleteNode(s.leafTempNodeWorkId, -1)
}
