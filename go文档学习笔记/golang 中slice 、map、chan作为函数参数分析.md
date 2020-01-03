写这篇文章之前考虑一个问题：
- [ ]     **go里面都是值传递，不存在引用传递？**
  
先来总结一下slice、map、chan的特性：  
**slice：**  
```
func makeslice64(et *_type, len64, cap64 int64) unsafe.Pointer

type slice struct {
	array unsafe.Pointer
	len   int
	cap   int
}

```
其实makeslice64返回是的[]int

1. slice本身是一个结构体，而不是一个指针，其底层实现是指向数组的指针。  
2. 三要素:type（指针）、len、cap

**map：**
```
func makemap(t *maptype, hint int, h *hmap) *hmap
```
1. makemap返回的一个指针
2. go map底层实现是hashmap，并采用链地址方法解决hash冲突的
3. map的扩容机制：2倍扩容，渐进式扩容。

# slice作为参数
先看看例子：

```
package main

import (
	"fmt"
	"reflect"
)

func modify1(slice []int)  {
	for i:=0;i<len(slice);i++{
		slice[i] = 0
	}

	fmt.Println("Inside  modify1 after append: ", len(slice))
	fmt.Println("Inside  modify1 after append: ", cap(slice))
	fmt.Println("Inside  modify1 after append: ", slice)
}

func modify2(slice []int)  {
	length := len(slice)
	for i:=0;i<length;i++{
		slice = append(slice, 1)
	}

	fmt.Println("Inside  modify2 after append: ", len(slice))
	fmt.Println("Inside  modify2 after append: ", cap(slice))
	fmt.Println("Inside  modify2 after append: ", slice)
}

func main(){

	s1 := make([]int,10,10)
	for i:=0;i<10;i++{
		s1[i] = i
	}
	fmt.Println("makeslcie return type: ", reflect.TypeOf(s1))

	fmt.Println("before modify slice: ", s1)

	modify1(s1)
	fmt.Println("after modify1 slice: ", s1)

    for i:=0;i<10;i++{
		s1[i] = i
	}
	modify2(s1)
	fmt.Println("after modify2 slice: ", len(s1))
	fmt.Println("after modify2 slice: ", cap(s1))
	fmt.Println("after modify2 slice: ", s1)

}
```
运行看看输出：

```
makeslcie return type:  []int
before modify slice:  [0 1 2 3 4 5 6 7 8 9]
Inside  modify1 after append:  10
Inside  modify1 after append:  10
Inside  modify1 after append:  [0 0 0 0 0 0 0 0 0 0]
after modify1 slice:  [0 0 0 0 0 0 0 0 0 0]
Inside  modify2 after append:  20
Inside  modify2 after append:  20
Inside  modify2 after append:  [0 1 2 3 4 5 6 7 8 9 1 1 1 1 1 1 1 1 1 1]
after modify2 slice:  10
after modify2 slice:  10
after modify2 slice:  [0 1 2 3 4 5 6 7 8 9]
```
modify1看，slice作为参数，在函数内部能修改slice，表面看确实能在函数内部修改slice  

modify2看，在函数modify2内部用appen操作扩容了slice，len和cap都变成了20，但是再看看后面的输出，modify2并没有修改slice，外部的slice依然没变 len和cap都没变化。  

**这是怎么回事，函数内部修改slice并没有影响外部slice。 其实go里面都是值传递，makeslice返回的是[]int，传入函数内部会对其拷贝一份，slice内部实现是指向数组的指针的，拷贝的副本部分底层实现也是指向同一内存地址的指针数组。所以内部修改slice的值是能修改的，但是append的并没有修改传入的slice的数组，而是返回一个新的slice的，这要去看看slice的实现和其append的扩容机制。实际上当函数内部不扩容slice，如果修改slice也是修改其指向的底层数组。如果发生扩容会发生数据拷贝，并不会修改其指向的array数组。**

**如果想在函数内部修改可以传递数组指针就可以了，类似下面这样**：

```
func modify2(slice *[]int)
```
参考资料：https://www.cnblogs.com/junneyang/p/6074786.html

