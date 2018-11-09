package snowflake

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"testing"
	"time"
)

func TestSnowMarshalLeafForeverNodeData(t *testing.T) {

	data := LeafForeverNodeData {
		Host: "192.168.0.106",
		Time: "2018-10-04 20:01:19.1718396 +0800 CST",
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("json marshal %v faild", data)
		panic(err)
	}



	fmt.Printf("data %v--> %s\n", data, string(bytes))

	var data2 LeafForeverNodeData
	err = json.Unmarshal(bytes, &data2)
	if err != nil {
		fmt.Printf("json unmarshal %s faild", string(bytes))
		panic(err)
	}

	fmt.Printf("data2 %s-> %v\n", string(bytes), data)
}

func TestSignal(t *testing.T) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	for {
		select {
		case sig := <-c:
			log.Printf("Got signal: %v, server exit!", sig)
		}
	}


}

// go test -bench . -cpuprofile=cpu.out
// go tool pprof cpu.out
// web
func BenchmarkSnowFlake_GenerateId(b *testing.B) {
	snow := SnowFlake{}

	b.Logf("b.N=%d", b.N)

	start := time.Now().UnixNano()

	for i := 0; i < b.N; i++ {
		snow.GenerateId()
	}

	end := time.Now().UnixNano()
	b.Logf("start %v, end %v, spent %v", start, end, end - start)
}



func Benchmark_ServeIdReq_FakeReq(b *testing.B) {
	b.Logf("b.N=%d", b.N)

	start := time.Now().UnixNano()

	snow := SnowFlake{}

	for i := 0; i < b.N; i++ {

		//创建一个请求
		req, err := http.NewRequest("GET", "/snowflake/id", nil)
		if err != nil {
			b.Fatal(err)
		}

		// 我们创建一个 ResponseRecorder (which satisfies http.ResponseWriter)来记录响应
		rr := httptest.NewRecorder()

		//直接使用HealthCheckHandler，传入参数rr,req
		snow.ServeIdReq(rr, req)

		//b.Log(rr.Body.String())

		// 检测返回的状态码
		if status := rr.Code; status != http.StatusOK {
			b.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	}
	end := time.Now().UnixNano()
	b.Logf("start %v, end %v, spent %v", start, end, end - start) // 52w/s

}


const testUrl = "http://127.0.0.1:8090/snowflake/id"


func Benchmark_ServeIdReq_RealReq(b *testing.B) {
	b.Logf("b.N=%d", b.N)

	start := time.Now().UnixNano()

	for i := 0; i < b.N; i++ {

		var client http.Client

		//创建一个请求
		req, err := http.NewRequest("GET", testUrl, nil)
		if err != nil {
			b.Fatal(err)
		}

		resp, err := client.Do(req)
		if err  != nil {
			panic(err)
		}
		// 检测返回的状态码
		if status := resp.StatusCode; status != http.StatusOK {
			b.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		resp.Body.Close()
	}
	end := time.Now().UnixNano()
	b.Logf("start %v, end %v, spent %v", start, end, end - start) // // 每个请求 0.46ms

}

func Benchmark_ServeIdReq_RealReq2(b *testing.B) {
	b.Logf("b.N=%d", b.N)

	start := time.Now().UnixNano()



	for i := 0; i < b.N; i++ {

		resp, err := http.Get(testUrl)
		if err != nil {
			panic(err)
			return
		}

		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		// 检测返回的状态码
		if status := resp.StatusCode; status != http.StatusOK {
			b.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		resp.Body.Close()
	}
	end := time.Now().UnixNano()
	b.Logf("start %v, end %v, spent %v", start, end, end - start) // 每个请求 0.16ms

}
