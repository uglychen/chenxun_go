# PREFIX_SIZE说明

在zmalloc函数中，实际可能会每次多申请一个 PREFIX_SIZE的空间。从如下的代码中看出，如果定义了宏HAVE_MALLOC_SIZE，那么 PREFIX_SIZE的长度为0。其他的情况下，都会多分配至少8字节的长度的内存空间
```
#ifdef HAVE_MALLOC_SIZE
#define PREFIX_SIZE (0)
#else
#if defined(__sun) || defined(__sparc) || defined(__sparc__)
#define PREFIX_SIZE (sizeof(long long))
#else
#define PREFIX_SIZE (sizeof(size_t))
#endif
#endif
```
**这么做的原因是因为： tcmalloc 和 Mac平台下的 malloc 函数族提供了计算已分配空间大小的函数（分别是tcmallocsize和mallocsize），所以就不需要单独分配一段空间记录大小了。而针对linux和sun平台则要记录分配空间大小。对于linux，使用sizeof(sizet)定长字段记录；对于sun os，使用sizeof(long long)定长字段记录。因此当宏HAVE_MALLOC_SIZE没有被定义的时候，就需要在多分配出的空间内记录下当前申请的内存空间的大小。**

根据情况使用不同的malloc实现  自行google USE_TCMALLOC和USE_JEMALLOC实现
```
/* Explicitly override malloc/free etc when using tcmalloc. */
#if defined(USE_TCMALLOC)
#define malloc(size) tc_malloc(size)
#define calloc(count,size) tc_calloc(count,size)
#define realloc(ptr,size) tc_realloc(ptr,size)
#define free(ptr) tc_free(ptr)
#elif defined(USE_JEMALLOC)
#define malloc(size) je_malloc(size)
#define calloc(count,size) je_calloc(count,size)
#define realloc(ptr,size) je_realloc(ptr,size)
#define free(ptr) je_free(ptr)
#define mallocx(size,flags) je_mallocx(size,flags)
#define dallocx(ptr,flags) je_dallocx(ptr,flags)
#endif

```
# update_zmalloc_stat_alloc 和 update_zmalloc_stat_free
这个函数的作用是跟新全局变量**used_memory**：redis内存使用情况

malloc函数本身能够保证分配的内存是8字节对齐的，如果要分配的内存不是8的倍数，那么malloc就会多分配一点，来凑成8的倍数。  

if (_n&(sizeof(long)-1)) _n += sizeof(long)-(_n&(sizeof(long)-1));这行代码是实现字节对齐， sizeof(long) = 8（64位系统， 32位系统中为4），这段代码就是判断分配的内存空间的大小是不是8的倍数。如果内存大小不是8的倍数，就加上相应的偏移量使之变成8的倍数
  
 _n&(sizeof(long)-1) ： _n&7 在功能上等价于 _n%8，位操作效率高  
 
 
 **atomicIncr**: redis自己实现原子操作函数：其内存实现也是调用系统底层原子操作接口，自行去学习__atomic 与 __sync两个版本的原子操作系统接口
 
```
#define update_zmalloc_stat_alloc(__n) do { \
    size_t _n = (__n); \
    if (_n&(sizeof(long)-1)) _n += sizeof(long)-(_n&(sizeof(long)-1)); \
    atomicIncr(used_memory,__n); \
} while(0)

#define update_zmalloc_stat_free(__n) do { \
    size_t _n = (__n); \
    if (_n&(sizeof(long)-1)) _n += sizeof(long)-(_n&(sizeof(long)-1)); \
    atomicDecr(used_memory,__n); \
} while(0)

static size_t used_memory = 0;
pthread_mutex_t used_memory_mutex = PTHREAD_MUTEX_INITIALIZER;

```

# zmalloc

```
void *zmalloc(size_t size) {
    //分配size+PREFIX_SIZE长度的内存
    void *ptr = malloc(size+PREFIX_SIZE);

    //报错
    if (!ptr) zmalloc_oom_handler(size);
#ifdef HAVE_MALLOC_SIZE
    
    //字节对齐 原子 更新全局变量used_memory
    update_zmalloc_stat_alloc(zmalloc_size(ptr));
    return ptr;
#else
    
    //如果未定义HAVE_MALLOC_SIZE，多分配的PREFIX_SIZE长度的内存存贮使用的内存大小 
    *((size_t*)ptr) = size;
    update_zmalloc_stat_alloc(size+PREFIX_SIZE);
    //向前移动指针ptr
    return (char*)ptr+PREFIX_SIZE;
#endif
}
```

# zmalloc_size
**通过指针ptr求内存使用的大小**
```
#ifndef HAVE_MALLOC_SIZE
size_t zmalloc_size(void *ptr) {
    void *realptr = (char*)ptr-PREFIX_SIZE;
    size_t size = *((size_t*)realptr);
    /* Assume at least that all the allocations are padded at sizeof(long) by
     * the underlying allocator. */
    if (size&(sizeof(long)-1)) size += sizeof(long)-(size&(sizeof(long)-1));
    return size+PREFIX_SIZE;
}
size_t zmalloc_usable(void *ptr) {
    return zmalloc_size(ptr)-PREFIX_SIZE;
}
#endif
```
# zfree释放内存
```
void zfree(void *ptr) {
#ifndef HAVE_MALLOC_SIZE
    void *realptr;
    size_t oldsize;
#endif

    if (ptr == NULL) return;
#ifdef HAVE_MALLOC_SIZE
    update_zmalloc_stat_free(zmalloc_size(ptr));
    free(ptr);
#else
    realptr = (char*)ptr-PREFIX_SIZE;
    oldsize = *((size_t*)realptr);
    update_zmalloc_stat_free(oldsize+PREFIX_SIZE);
    free(realptr);
#endif
}
```
**可以发现如果使用的libc库，则需要将ptr指针向前偏移8个字节的长度，回退到最初malloc返回的地址，然后通过类型转换再取指针所指向的值。通过zmalloc()函数的分析，可知这里存储着我们最初需要分配的内存大小（zmalloc中的size），这里赋值给oldsize。update_zmalloc_stat_free()也是一个宏函数，和zmalloc中update_zmalloc_stat_alloc()大致相同，唯一不同之处是前者在给变量used_memory减去分配的空间，而后者是加上该空间大小。
最后free(realptr)，清除空间**。

# zcalloc
- zcalloc 函数与zmalloc函数的功能基本相同，但有2点不同的是:
- 分配的空间大小是 size * nmemb。;
- calloc()会对分配的空间做初始化工作（初始化为0），而malloc()不会。
```
void *zcalloc(size_t size) {
    void *ptr = calloc(1, size+PREFIX_SIZE);

    if (!ptr) zmalloc_oom_handler(size);
#ifdef HAVE_MALLOC_SIZE
    update_zmalloc_stat_alloc(zmalloc_size(ptr));
    return ptr;
#else
    *((size_t*)ptr) = size;
    update_zmalloc_stat_alloc(size+PREFIX_SIZE);
    return (char*)ptr+PREFIX_SIZE;
#endif
}
```
# zrealloc 
**调整内存大小：**
zrealloc 函数用来修改内存大小。具体的流程基本是分配新的内存大小，然后把老的内存数据拷贝过去，之后释放原有的内存。
```
void *zrealloc(void *ptr, size_t size) {
#ifndef HAVE_MALLOC_SIZE
    void *realptr;
#endif
    size_t oldsize;
    void *newptr;

    if (ptr == NULL) return zmalloc(size);
#ifdef HAVE_MALLOC_SIZE
    oldsize = zmalloc_size(ptr);
    newptr = realloc(ptr,size);
    if (!newptr) zmalloc_oom_handler(size);

    update_zmalloc_stat_free(oldsize);
    update_zmalloc_stat_alloc(zmalloc_size(newptr));
    return newptr;
#else
    realptr = (char*)ptr-PREFIX_SIZE;
    oldsize = *((size_t*)realptr);
    newptr = realloc(realptr,size+PREFIX_SIZE);
    if (!newptr) zmalloc_oom_handler(size);

    *((size_t*)newptr) = size;
    update_zmalloc_stat_free(oldsize+PREFIX_SIZE);
    update_zmalloc_stat_alloc(size+PREFIX_SIZE);
    return (char*)newptr+PREFIX_SIZE;
#endif
}
```
# zstrdup
redis的对memcpy的封装
```
char *zstrdup(const char *s) {
    size_t l = strlen(s)+1;
    char *p = zmalloc(l);

    memcpy(p,s,l);
    return p;
}
```

# zmalloc_get_rss
这个函数可以获取当前进程实际所驻留在内存中的空间大小，即不包括被交换（swap）出去的空间。该函数大致的操作就是在当前进程的 /proc//stat 【表示当前进程id】文件中进行检索。该文件的第24个字段是RSS的信息，它的单位是pages（内存页的数目)。如果没从操作系统的层面获取驻留内存大小，那就只能绌劣的返回已经分配出去的内存大小。
```
/* Get the RSS information in an OS-specific way.
 *
 * WARNING: the function zmalloc_get_rss() is not designed to be fast
 * and may not be called in the busy loops where Redis tries to release
 * memory expiring or swapping out objects.
 *
 * For this kind of "fast RSS reporting" usages use instead the
 * function RedisEstimateRSS() that is a much faster (and less precise)
 * version of the function. */

#if defined(HAVE_PROC_STAT)
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>

size_t zmalloc_get_rss(void) {
    int page = sysconf(_SC_PAGESIZE);
    size_t rss;
    char buf[4096];
    char filename[256];
    int fd, count;
    char *p, *x;

    snprintf(filename,256,"/proc/%d/stat",getpid());
    if ((fd = open(filename,O_RDONLY)) == -1) return 0;
    if (read(fd,buf,4096) <= 0) {
        close(fd);
        return 0;
    }
    close(fd);

    p = buf;
    count = 23; /* RSS is the 24th field in /proc/<pid>/stat */
    while(p && count--) {
        p = strchr(p,' ');
        if (p) p++;
    }
    if (!p) return 0;
    x = strchr(p,' ');
    if (!x) return 0;
    *x = '\0';

    rss = strtoll(p,NULL,10);
    rss *= page;
    return rss;
}
#elif defined(HAVE_TASKINFO)
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/sysctl.h>
#include <mach/task.h>
#include <mach/mach_init.h>

size_t zmalloc_get_rss(void) {
    task_t task = MACH_PORT_NULL;
    struct task_basic_info t_info;
    mach_msg_type_number_t t_info_count = TASK_BASIC_INFO_COUNT;

    if (task_for_pid(current_task(), getpid(), &task) != KERN_SUCCESS)
        return 0;
    task_info(task, TASK_BASIC_INFO, (task_info_t)&t_info, &t_info_count);

    return t_info.resident_size;
}
#else
size_t zmalloc_get_rss(void) {
    /* If we can't get the RSS in an OS-specific way for this system just
     * return the memory usage we estimated in zmalloc()..
     *
     * Fragmentation will appear to be always 1 (no fragmentation)
     * of course... */
    return zmalloc_used_memory();
}
#endif
```
