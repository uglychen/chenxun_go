参考文章：  
https://studygolang.com/articles/18194?fr=sidebar  
https://github.com/cch123/golang-notes/blob/master/slice.md  
https://blog.haohtml.com/archives/18094

# slice:
先看一下slcie结构：
```
runtime/slice.go
type slice struct {
    array unsafe.Pointer
    len   int
    cap   int
}
```
**slice三要素： type、len、cap**

slice 的底层结构定义非常直观，指向底层数组的指针，当前长度 len 和当前 slice 的 cap。

                                                                   
       []int{1,3,4,5}                                              
                                                                   
       struct {                                                    
         array unsafe.Pointer --------------+                      
         len int                            |                      
         cap int                            |                      
       }                                    |                      
                                            |                      
                                            v                      
                                                                   
                               +------|-------|------|------+-----+
                               |      |  1    |  3   | 4    |  5  |
                               |      |       |      |      |     |
                               +------|-------|------|------+-----+
                                                 [5]int            
                                                         
                                                         
                                                        
                                                        
                                                        

```
func makeslice(et *_type, len, cap int) slice
```

# slice扩容机制：
扩容时会判断 slice 的 cap 是不是已经大于 1024，如果在 1024 之内，会按二倍扩容。超过的话就是 1.25 倍扩容了。

slice 扩容必然会导致内存拷贝，如果是性能敏感的系统中，尽可能地提前分配好 slice 是较好的选择。
```
var arr = make([]int, 0, 10)
```

```
func growslice(et *_type, old slice, cap int) slice {

    if et.size == 0 {
        if cap < old.cap {
            panic(errorString("growslice: cap out of range"))
        }
        return slice{unsafe.Pointer(&zerobase), old.len, cap}
    }

    newcap := old.cap
    doublecap := newcap + newcap
    if cap > doublecap {
        newcap = cap
    } else {
        // 注意这里的 1024 阈值
        if old.len < 1024 {
            newcap = doublecap
        } else {
            // Check 0 < newcap to detect overflow
            // and prevent an infinite loop.
            for 0 < newcap && newcap < cap {
                newcap += newcap / 4
            }
            // Set newcap to the requested cap when
            // the newcap calculation overflowed.
            if newcap <= 0 {
                newcap = cap
            }
        }
    }

    var overflow bool
    var lenmem, newlenmem, capmem uintptr
    const ptrSize = unsafe.Sizeof((*byte)(nil))
    switch et.size {
    case 1:
        lenmem = uintptr(old.len)
        newlenmem = uintptr(cap)
        capmem = roundupsize(uintptr(newcap))
        overflow = uintptr(newcap) > _MaxMem
        newcap = int(capmem)
    case ptrSize:
        lenmem = uintptr(old.len) * ptrSize
        newlenmem = uintptr(cap) * ptrSize
        capmem = roundupsize(uintptr(newcap) * ptrSize)
        overflow = uintptr(newcap) > _MaxMem/ptrSize
        newcap = int(capmem / ptrSize)
    default:
        lenmem = uintptr(old.len) * et.size
        newlenmem = uintptr(cap) * et.size
        capmem = roundupsize(uintptr(newcap) * et.size)
        overflow = uintptr(newcap) > maxSliceCap(et.size)
        newcap = int(capmem / et.size)
    }

    if cap < old.cap || overflow || capmem > _MaxMem {
        panic(errorString("growslice: cap out of range"))
    }

    var p unsafe.Pointer
    if et.kind&kindNoPointers != 0 {
        p = mallocgc(capmem, nil, false)
        memmove(p, old.array, lenmem)
        // The append() that calls growslice is going to overwrite from old.len to cap (which will be the new length).
        // Only clear the part that will not be overwritten.
        memclrNoHeapPointers(add(p, newlenmem), capmem-newlenmem)
    } else {
        // Note: can't use rawmem (which avoids zeroing of memory), because then GC can scan uninitialized memory.
        p = mallocgc(capmem, et, true)
        if !writeBarrier.enabled {
            memmove(p, old.array, lenmem)
        } else {
            for i := uintptr(0); i < lenmem; i += et.size {
                typedmemmove(et, add(p, i), add(old.array, i))
            }
        }
    }

    return slice{p, old.len, newcap}
}
```


# slice的append()函数

```
func append(slice []Type, elems ...Type) []Type
```
**函数说明：内建函数append追加一个或多个elems到一个slice依赖的array的末尾，如果这个slice有足够的capacity，则reslice以容纳新增元素；如果capacity空间不够，则进行扩容，重新分配内存保存新的slice依赖的array，函数返回更新后的slice.**

**注意：append不会修改传参进来的slice(len和cap)，只会在不够用的时候新分配一个array，并把之前的slice依赖的array数据拷贝过来；所以对同一个slice 重复 append，只要不超过cap，都是修改的同一个array，后面的会覆盖前面**


# slice是做函数参数是值传递：

```
func main() {
    var a = make([]int, 10)
    fmt.Println(a)
}

func doSomeHappyThings(sl []int) {
    if len(sl) > 0 {
        sl[0] = 1
    }
}
```
把 a 传入到 doSomeHappyThings，然后 a 的第一个元素就被修改了，进而认为在 Go 中，slice 是引用传递的。其实这是错误，go里面都是值传递，不存在引用传递。



