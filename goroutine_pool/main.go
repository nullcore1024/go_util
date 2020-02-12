package main

import (
	"./pool"
	"fmt"
	"time"
)

func main() {

	var wf pool.WorkFlow
	//初始化并启动工作
	wf.StartWorkFlow(2, 4)
	for i := 0; i < 100; i++ {
		payload := pool.Payload{
			fmt.Sprintf("产品-%08d", i+1),
		}
		wJob := pool.Job{
			Payload: payload,
		}
		//添加å¥作
		wf.AddJob(wJob)
		//time.Sleep(time.Millisecond * 10)

	}
	wf.CloseWorkFlow()

	time.Sleep(time.Second * 20)
}
