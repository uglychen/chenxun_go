参考文章：
[CAS（Compare and Swap）算法介绍、缺陷和解决思路](  
https://blog.csdn.net/q2878948/article/details/90105951  
https://www.jianshu.com/p/c74c85db5129)

# CAS（compare and swap）  
1. go中CAS操作具有原子性，在解决多线程操作共享变量安全上可以有效的减少使用锁所带来的开销，但是这是使用cpu资源做交换的  
1.  go中的Cas操作与java中类似，都是借用了CPU提供的原子性指令来实现。CAS操作修改共享变量时候不需要对共享变量加锁，而是通过类似乐观锁的方式进行检查，本质还是不断的占用CPU 资源换取加锁带来的开销（比如上下文切换开销）（参考文章：https://www.jianshu.com/p/4e61ed8e140a）


**原子操作主要由硬件提供支持，锁一般是由操作系统提供支持，比起直接使用锁，使用CAS这个过程不需要形成临界区和创建互斥量，所以会比使用锁更加高效。**

从硬件层面来实现原子操作，有两种方式：

1、总线加锁：因为CPU和其他硬件的通信都是通过总线控制的，所以可以通过在总线加LOCK#锁的方式实现原子操作，但这样会阻塞其他硬件对CPU的访问，开销比较大。

2、缓存锁定：频繁使用的内存会被处理器放进高速缓存中，那么原子操作就可以直接在处理器的高速缓存中进行而不需要使用总线锁，主要依靠缓存一致性来保证其原子性。
————————————————
版权声明：本文为CSDN博主「菌菇」的原创文章，遵循 CC 4.0 BY-SA 版权协议，转载请附上原文出处链接及本声明。
原文链接：https://blog.csdn.net/qq_39920531/article/details/97646901

**下面一个例子使用CAS来实现计数器，把这个例子理解了差不多就理解cas原理了**:
```
package main

import (
"fmt"
"sync"
"sync/atomic"
)

var (
	counter int32//计数器
	wg sync.WaitGroup //信号量
)

func main() {
	threadNum := 5 //1. 五个信号量
	wg.Add(threadNum) //2.开启5个线程
	for i := 0; i < threadNum; i++ {
		go incCounter(i)
	}
	//3.等待子线程结束
	wg.Wait()
	fmt.Println(counter)
}

func incCounter(index int) {
	defer wg.Done()
	spinNum := 0
	for {
		//2.1原子操作
		old := counter
		ok := atomic.CompareAndSwapInt32(&counter, old, old+1)
		if ok {
			break
		} else {
			spinNum++
		}
	}
	fmt.Printf("thread,%d,spinnum,%d\n",index,spinNum)
}
```
执行结果
```
thread,4,spinnum,0
thread,0,spinnum,0
thread,1,spinnum,0
thread,3,spinnum,0
thread,2,spinnum,0
5

```

如上代码main线程首先创建了5个信号量，然后开启五个线程执行incCounter方法

incCounter内部执行代码2.1 使用cas操作递增counter的值， atomic.CompareAndSwapInt32具有三个参数，第一个是变量的地址，第二个是变量当前值，第三个是要修改变量为多少，该函数如果发现传递的old值等于当前变量的值，则使用第三个变量替换变量的值并返回true，否则返回false。

这里之所以使用无限循环是因为在高并发下每个线程执行CAS并不是每次都成功，失败了的线程需要重写获取变量当前的值，然后重新执行CAS操作。读者可以把线程数改为10000或者更多会发现输出thread,5329,spinnum,1其中1说明该线程尝试了两个CAS操作，第二次才成功。

# CAS的缺陷
1.循环开销大  
可以看到，方法内部用不断循环的方式实现修改。如果CAS长时间一直不成功，可能会给CPU带来很大的开销。

2.只能保证一个共享变量的原子操作  
需要对多个共享变量操作时，循环CAS就无法保证操作的原子性。  
解决方法：可以把多个变量放在一个对象里来进行CAS操作。

3.ABA问题  
CAS需要在操作值的时候，检查值有没有发生变化，如果没有发生变化则更新，但是如果一个值原来是A，变成了B，又变成了A，那么CAS进行检查的时候发现它的值没有发生变化，但是实质上它已经发生了改变 。可能会造成数据的缺失

有篇文章:[Go 的一个 CAS 操作使用场景](https://segmentfault.com/a/1190000020276792)可以看一看。

这个例子是在使用cas代替互斥锁：（降低开销）

```
package main

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

const (
	_CHAN_SIZE  = 10
	_GUARD_SIZE = 10

	_TEST_CNT = 32
)

type Obj struct {
	flag int64
	c    chan interface{}
}

func (obj *Obj) readLoop() error {
	counter := _TEST_CNT
	for {
		time.Sleep(5 * time.Millisecond)
		if len(obj.c) > _CHAN_SIZE {
			return errors.New(fmt.Sprintf("Chan overflow, len: %v.", len(obj.c)))
		} else if len(obj.c) > 0 {
			<-obj.c
			counter--
		}
		if counter <= 0 {
			return nil
		}
	}
}

func (obj *Obj) writeMsg(idx int, v interface{}) (err error) {
	for {
		if len(obj.c) < _CHAN_SIZE {
			obj.c <- v
			fmt.Printf("R(%v)+1 ", idx)
			return nil
		}
	}
}

func (obj *Obj) writeMsgWithCASCheck(idx int, v interface{}) (err error) {
	for {
		if atomic.CompareAndSwapInt64(&obj.flag, 0, 1) {
			if len(obj.c) < _CHAN_SIZE {
				obj.c <- v
				atomic.StoreInt64(&obj.flag, 0)
				fmt.Printf("R(%v)+1 ", idx)
				return nil
			} else {
				atomic.StoreInt64(&obj.flag, 0)
			}
		}
	}

	return nil
}

func main() {
	useCAS := false
	if len(os.Args) > 1 && os.Args[1] == "cas" {
		useCAS = true
	}
	routineCnt := 4
	tryCnt := _TEST_CNT / routineCnt
	var obj = &Obj{c: make(chan interface{}, _CHAN_SIZE+_GUARD_SIZE)}

	for idx := 0; idx < routineCnt; idx++ {
		go func(nameIdx int) {
			for tryIdx := 0; tryIdx < tryCnt; tryIdx++ {
				if useCAS {
					obj.writeMsgWithCASCheck(nameIdx, nil)
				} else {
					obj.writeMsg(nameIdx, nil)
				}
			}
		}(idx)
	}

	// fmt.Println(casObj.readLoop())
	fmt.Println(obj.readLoop())
	fmt.Println("quit.")
}
```
