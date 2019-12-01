package main

import (
    "time"
    "fmt"
)

func main() {

    a :=make(chan string)
    go sendDataTo(a)
    go timing()

    getAchan(10*time.Second,a)

}

func sendDataTo(a chan string)  {
    for {
        a <- "我是a通道的数据1111"
        time.Sleep(1e9 *1)
    }
}

//在一定时间内接收不到a的数据则超时
func getAchan(timeout time.Duration, a chan string)  {
    var after <-chan time.Time
loop:
    after = time.After(timeout)
    for{
        fmt.Println("等待a中的数据，10秒后没有数据则超时")
        select {
        case x :=<- a:
            fmt.Println("收到a通道中的数据" + x)
            goto loop
        case <-after:
            fmt.Println("timeout.")
            return
        }
    }
}
func timing()  {
    //定时器，10秒钟执行一次
    ticker := time.NewTicker(10 * time.Second)
    for {
        time := <-ticker.C
        fmt.Println("定时器====>",time.String())
    }
}
