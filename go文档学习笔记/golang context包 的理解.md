# **先看一下context包的英文介绍：**

```
// Programs that use Contexts should follow these rules to keep interfaces
// consistent across packages and enable static analysis tools to check context
// propagation:
//
// Do not store Contexts inside a struct type; instead, pass a Context
// explicitly to each function that needs it. The Context should be the first
// parameter, typically named ctx:
//
// 	func DoSomething(ctx context.Context, arg Arg) error {
// 		// ... use ctx ...
// 	}
//
// Do not pass a nil Context, even if a function permits it. Pass context.TODO
// if you are unsure about which Context to use.
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
//
// The same Context may be passed to functions running in different goroutines;
// Contexts are safe for simultaneous use by multiple goroutines.
//
// See https://blog.golang.org/context for example code for a server that uses
// Contexts.
```
其大概意思在介绍context的使用规则：
1. Context 变量需要作为第一个参数使用，一般命名为ctx。不要把 Context 存在一个结构体当中
2. 即使方法允许，也不要传入一个 nil 的 Context ，如果你不确定你要用什么 Context 的时候传一个 context.TODO
3. 使用 context 的 Value 相关方法只应该用于在程序和接口中传递的和请求相关的元数据，不要用它来传递一些可选的参数
4. 同样的 Context 可以用来传递到不同的 goroutine 中，Context 在多个goroutine 中是安全的

# 使用context的有几个方法：

```
func WithValue(parent Context, key interface{}, val interface{}) Context

func WithCancel(parent Context) (ctx Context, cancel CancelFunc)

func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc)

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)

```
**Context接口**
```
type Context interface {
	
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}
```

创建 context：  
```
func Background() Context {
	return background
}
```
**ctx := context.Background()**  
这个函数返回一个空 context。这只能用于高等级（在 main 或顶级请求处理中）。这能用于派生我们稍后谈及的其他 context 。

```
func TODO() Context {
	return todo
}
```
**context.TODO() Context  **  
这个函数也是创建一个空 context。也只能用于高等级或当您不确定使用什么 context，或函数以后会更新以便接收一个context。这意味您（或维护者）计划将来要添加 context 到函数。  
**查看源码background和todo是一样的：**
```
var (
    background = new(emptyCtx)
    todo = new(emptyCtx)
)
```

接下来我们逐一分析context的四个方法：（WithValue、WithCancel、WithDeadline、WithTimeout）
# **一、WithValue** 
1、可以把需要的信息放到context中，需要时把变量取出来  

2、WithValue方法获取该Context上绑定的值，是一个键值对，所以要通过一个Key才可以获取对应的值，这个值一般是线程安全的。但使用这些数据的时候要注意同步，比如返回了一个map，而这个map的读写则要加锁

context包中 **WithValue** 方法定义：
```
func WithValue(parent Context, key, val interface{}) Context {
	if key == nil {
		panic("nil key")
	}
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}
```
看例子：

```
package main

import (

	"fmt"
	"context"
)

func process(ctx context.Context) {
	ret,ok := ctx.Value("trace_id").(int)
	if !ok {
		ret = 21342423
	}

	fmt.Printf("ret:%d\n", ret)

	s , _ := ctx.Value("session").(string)
	fmt.Printf("session:%s\n", s)
}

func main() {
	ctx := context.WithValue(context.Background(), "trace_id", 2222222)
	ctx = context.WithValue(ctx, "session", "sdlkfjkaslfsalfsafjalskfj")
	process(ctx)
}
```

# 二、withCancel

```
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) 
```

WithCancel返回一个继承的Context,这个Context在父Context的Done被关闭时关闭自己的Done通道，或者在自己被Cancel的时候关闭自己的Done。WithCancel同时还返回一个取消函数cancel，这个cancel用于取消当前的Context

这是它开始变得有趣的地方。此函数创建从传入的父 context 派生的新 context。父 context 可以是后台 context 或传递给函数的 context。

返回派生 context 和取消函数。只有创建它的函数才能调用取消函数来取消此 context。如果您愿意，可以传递取消函数，但是，强烈建议不要这样做。这可能导致取消函数的调用者没有意识到取消 context 的下游影响。可能存在源自此的其他 context，这可能导致程序以意外的方式运行。简而言之，永远不要传递取消函数

```
package main

import (

	"fmt"
	"context"
	"time"
)


func test_withCancel(ctx context.Context, intChan chan int) {

	n := 1
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("test_withCancel process exited")
				return
			case intChan <- n:
				n++
			}
		}
	}()
}

func main() {

	intChan := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())

	go test_withCancel(ctx, intChan)

	{
		for elem := range intChan {
			fmt.Println(elem)

			if elem == 10{
				cancel() //发送通知 取消test_withCancel协程
				break
			}
		}
	}
	
	time.Sleep(time.Second*10)
}


```

# 三、withDeadline

```
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
```

deadline保存了超时的时间,   当超过这个时间,  会触发cancel, 如果超过了过期时间, 会自动撤销它的子context

此函数返回其父项的派生 context，当截止日期超过或取消函数被调用时，该 context 将被取消。例如，您可以创建一个将在以后的某个时间自动取消的 context，并在子函数中传递它。当因为截止日期耗尽而取消该 context 时，获此 context 的所有函数都会收到通知去停止运行并返回。


```
func main() {
	//deadline保存了超时的时间，当超过这个时间，会触发cancel,
	//如果超过了过期时间，会自动撤销它的子context
	d := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), d)

	// Even though ctx will be expired, it is good practice to call its
	// cancelation function in any case. Failure to do so may keep the
	// context and its parent alive longer than necessary.
	defer cancel()

	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Println("overslept")
		case <-ctx.Done():
			fmt.Println(ctx.Err())
			return
		}
	}
}
```


# 四、WithTimeout

```
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
```

可以用来控制goroutine超时，context包中提供的WithTimeout(本质上调用的是WithDeadline) 方法


```
func main() {

	// Pass a context with a timeout to tell a blocking function that it
	// should abandon its work after the timeout elapses.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(1 * time.Second):
		fmt.Println("overslept")
	case <-ctx.Done():
		fmt.Println(ctx.Err()) // prints "context deadline exceeded"
	}

}
```

https://studygolang.com/articles/13866?fr=sidebar文中有一个例子。我们来分析一下加深对context的理解：  

**main 函数**  
- 用 cancel 创建一个 context  
- 随机超时后调用取消函数  

**doWorkContext 函数**
- 派生一个超时 context 
- 这个 context 将被取消当  
  要么是 main 调用取消函数   
  或者就是 超时到时间了 doWorkContext 调用它的取消函数
- 启动 goroutine 传入派生上下文执行一些慢处理
- 等待 goroutine 完成或上下文被 main goroutine 取消，以优先发生者为准  

**sleepRandomContext 函数**

- 开启一个 goroutine 去做些缓慢的处理
- 等待该 goroutine 完成或，
- 等待 context 被 main goroutine 取消，操时或它自己的取消函数被调用

**sleepRandom 函数**

- 随机时间休眠
- 此示例使用休眠来模拟随机处理时间，在实际示例中，您可以使用通道来通知此函数，以开始清理并在通道上等待它，以确认清理已完成。

自己多运行几次下面的代码，查看日志，分析分析代码逻辑理解context

pis： 把main函数稍微修改再分析分析  

```
    go doWorkContext(ctxWithCancel)   
    time.Sleep(time.Second*2)
```



```
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

//Slow function
func sleepRandom(fromFunction string, ch chan int) {
	//defer cleanup
	defer func() { fmt.Println(fromFunction, "sleepRandom complete") }()
	//Perform a slow task
	//For illustration purpose,
	//Sleep here for random ms
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	randomNumber := r.Intn(100)
	sleeptime := randomNumber + 100
	fmt.Println(fromFunction, "Starting sleep for", sleeptime, "ms")
	time.Sleep(time.Duration(sleeptime) * time.Millisecond)
	fmt.Println(fromFunction, "Waking up, slept for ", sleeptime, "ms")
	//write on the channel if it was passed in
	if ch != nil {
		ch <- sleeptime
	}
}

//Function that does slow processing with a context
//Note that context is the first argument
func sleepRandomContext(ctx context.Context, ch chan bool) {
	//Cleanup tasks
	//There are no contexts being created here
	//Hence, no canceling needed
	defer func() {
		fmt.Println("6666666666sleepRandomContext complete")
		ch <- true
	}()
	//Make a channel
	sleeptimeChan := make(chan int)
	//Start slow processing in a goroutine
	//Send a channel for communication
	go sleepRandom("22222222222sleepRandomContext", sleeptimeChan)
	//Use a select statement to exit out if context expires
	select {
	case <-ctx.Done():
		//If context is cancelled, this case is selected
		//This can happen if the timeout doWorkContext expires or
		//doWorkContext calls cancelFunction or main calls cancelFunction
		//Free up resources that may no longer be needed because of aborting the work
		//Signal all the goroutines that should stop work (use channels)
		//Usually, you would send something on channel,
		//wait for goroutines to exit and then return
		//Or, use wait groups instead of channels for synchronization
		fmt.Println("sleepRandomContext: 444444444444444444Time to return")
	case sleeptime := <-sleeptimeChan:
		//This case is selected when processing finishes before the context is cancelled
		fmt.Println("888888Slept for ", sleeptime, "4444ms")
	}
}

//A helper function, this can, in the real world do various things.
//In this example, it is just calling one function.
//Here, this could have just lived in main
func doWorkContext(ctx context.Context) {
	//Derive a timeout context from context with cancel
	//Timeout in 150 ms
	//All the contexts derived from this will returns in 150 ms
	ctxWithTimeout, cancelFunction := context.WithTimeout(ctx, time.Duration(150)*time.Millisecond)
	//Cancel to release resources once the function is complete
	defer func() {
		fmt.Println("doWorkContext complete")
		cancelFunction()
	}()
	//Make channel and call context function
	//Can use wait groups as well for this particular case
	//As we do not use the return value sent on channel
	ch := make(chan bool)
	go sleepRandomContext(ctxWithTimeout, ch)
	//Use a select statement to exit out if context expires
	select {
	case <-ctx.Done():
		//This case is selected when the passed in context notifies to stop work
		//In this example, it will be notified when main calls cancelFunction
		fmt.Println("33333333doWorkContext: Time to return")
	case <-ch:
		//This case is selected when processing finishes before the context is cancelled
		fmt.Println("55555555555sleepRandomContext returned")
	}
}
func main() {
	//Make a background context
	ctx := context.Background()
	//Derive a context with cancel
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)
	//defer canceling so that all the resources are freed up
	//For this and the derived contexts
	defer func() {
		fmt.Println("Main Defer: canceling context")
		cancelFunction()
	}()
	//Cancel context after a random time
	//This cancels the request after a random timeout
	//If this happens, all the contexts derived from this should return
	go func() {
		sleepRandom("111111Main", nil)
		cancelFunction()
		fmt.Println("Main Sleep complete. canceling context")
	}()
	//Do work
	doWorkContext(ctxWithCancel)
}
```
