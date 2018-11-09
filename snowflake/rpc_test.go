package snowflake

import (
	"testing"
	"net/rpc/jsonrpc"
	"net"
	"net/rpc"
	"time"
)

func TestTimeService_GetSysTime(t *testing.T) {

	var req TimeServiceReq
	var rsp TimeServiceRsp

	const host = ":1234"

	// server
	listener, err := net.Listen("tcp", host)
	if err != nil {
		panic(err)
	}

	rpc.Register(TimeService{})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			jsonrpc.ServeConn(conn)
		}
	}()

	// client
	client, err := jsonrpc.Dial("tcp", host)
	if err != nil {
		panic(err)
	}

	err = client.Call(TimeServiceMethod, req, &rsp)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	t.Logf("now %v, recv server time %+v", now, rsp.Time)

}
