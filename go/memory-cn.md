内存管理
-------------

内存分配器中的对象有2种:

* span: 多个地址连续的`page`组成的大块内存
* object: 将span按照特定大小且分成小块，每个小块存储一个对象

栈、内存分配器、调度器相关初始化入口：

```
// src/runtime/proc.go
func schedinit() {
	sched.maxmcount = 10000
  // ...
  // 初始化内存模型，以后再看具体实现
	stackinit()
	mallocinit()
	mcommoninit(_g_.m)

  // ...
}
```

内存分配算法实现:

```
func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {
	if gcphase == _GCmarktermination {
		throw("mallocgc called with gcphase == _GCmarktermination")
	}

	if size == 0 {
		return unsafe.Pointer(&zerobase)
	}

	if debug.sbrk != 0 {
		align := uintptr(16)
		if typ != nil {
			align = uintptr(typ.align)
		}
		return persistentalloc(size, align, &memstats.other_sys)
	}

	// assistG is the G to charge for this allocation, or nil if
	// GC is not currently active.
	var assistG *g
	if gcBlackenEnabled != 0 {
		// Charge the current user G for this allocation.
		assistG = getg()
		if assistG.m.curg != nil {
			assistG = assistG.m.curg
		}
		// Charge the allocation against the G. We'll account
		// for internal fragmentation at the end of mallocgc.
		assistG.gcAssistBytes -= int64(size)

		if assistG.gcAssistBytes < 0 {
			// This G is in debt. Assist the GC to correct
			// this before allocating. This must happen
			// before disabling preemption.
			gcAssistAlloc(assistG)
		}
	}

	// Set mp.mallocing to keep from being preempted by GC.
	mp := acquirem()
	if mp.mallocing != 0 {
		throw("malloc deadlock")
	}
	if mp.gsignal == getg() {
		throw("malloc during signal")
	}
	mp.mallocing = 1

	shouldhelpgc := false
	dataSize := size
	// 获取当前goroutine的local cache
	c := gomcache()
	var x unsafe.Pointer
	// typ.kind&kindNoPointers => 判断非指针类型
	noscan := typ == nil || typ.kind&kindNoPointers != 0
	// 处理小对象: maxSmallSize => 32 << 10, 32 KB
	if size <= maxSmallSize {
		// 处理微对象: maxTinySize => 16 B
		if noscan && size < maxTinySize {
			// 16B小对象偏移量, 0 - 15
			// [*][*][*][*][][][][][][][][][][][][]
			// off = 3
			off := c.tinyoffset
			// Align tiny pointer for required (conservative) alignment.
			// [*][*][*][*][][][][][][][][][][][][]
			// 比如要分配一个4字节的空间

			// 这里的内存对齐貌似已经没什么用了，因为输入和输出是一样的
			if size&7 == 0 {
				// size是8的整数倍
				off = round(off, 8)
			} else if size&3 == 0 {
				// size是4的整数倍
				off = round(off, 4)
			} else if size&1 == 0 {
				// size是2的整数倍
				off = round(off, 2)
			}
			// 走到这里，off还是4
			// 这个时候如果发现off+size < maxTinySize
			// 就是说当前的16B cache里面有空间可以容纳4B的微对象
			// c.tiny != 0的意思是说当前的cache中是存在有效的内存空间的
			if off+size <= maxTinySize && c.tiny != 0 {
				// The object fits into existing tiny block.
				// c.tiny指向上一次分配的开始地址，
				// 这一次分配就从上一次分配开始地址＋对齐后的offset开始分配
				x = unsafe.Pointer(c.tiny + off)
				// c.tinyoffset是cache中可以继续往下分配的相对位置
				c.tinyoffset = off + size
				c.local_tinyallocs++
				mp.mallocing = 0
				// 这里的加锁解锁似乎不是很严谨
				releasem(mp)
				return x
			}
			// Allocate a new maxTinySize block.
			// 走到这里说明当前的cache里面空间不足, 或者cache没有初始化
			// local cache里面拿一个span出来
			span := c.alloc[tinySizeClass]
			// 从span里面获取一个可用的object: 16B
			v := nextFreeFast(span)
			// 如果v == 0，说明没有可用的object
			if v == 0 {
				// 发生这种情况可能是span上没有可用规格的内存页了
				// 具体的实现还没看，应该涉及一些大内存块的切分操作等
				v, _, shouldhelpgc = c.nextFree(tinySizeClass)
			}
			// 无论怎么样走到这一步就可以保证获取到一个有效的object
			x = unsafe.Pointer(v)
			// 一共16B，这里是把这16B分成2个uint64进行清零
			// 16 bytes => uint64 + uint64
			(*[2]uint64)(x)[0] = 0
			(*[2]uint64)(x)[1] = 0
			// See if we need to replace the existing tiny block with the new one
			// based on amount of remaining free space.
			// 终于看懂了！获取到新的object之后，并且为当前的对象在它上面分配了
			// 一块内存，那要不要用这个object来替换当前goroutine的微对象cache?
			// 决定于2个object剩余空间的大小
			// size是新获取的object的offset值，而goroutine当前的cache object内的
			// offset值为c.tinyoffset, size < c.tinyoffset说明新的剩余空间大
			// 或者c.tiny说明当前的goroutine还没有有效的cache object
			if size < c.tinyoffset || c.tiny == 0 {
				c.tiny = uintptr(x)
				c.tinyoffset = size
			}
			// 走到这一步说明待分配的对象是比8肯定要大了，而且不是8|4|2的整数倍
			// 直接把size设成maxTinySize简化内存管理的难度，会产生碎片
			size = maxTinySize
		} else {
			// 走到这一步说明待分配内存的小对象比较大，> 16B
			var sizeclass int8
			// 根据size计算sizeclass
			if size <= 1024-8 {
				// size: [1, 1024-8]
				// (size+7)>>3，结果是ceil(size/8)
				sizeclass = size_to_class8[(size+7)>>3]
			} else {
				// size > 1024-8
				// (size-1024+127)>>7，结果是ceil(size/128)
				sizeclass = size_to_class128[(size-1024+127)>>7]
			}
			// 对这样的对象，直接按照标准规格分配内存块, 直接把size设成标准大小
			size = uintptr(class_to_size[sizeclass])
			// 尝试从local cache获取span
			span := c.alloc[sizeclass]
			// 尝试从span上获取一个有效内存块
			v := nextFreeFast(span)
			if v == 0 {
				// 没有有效内存块，通过全局获取可用内存块
				v, span, shouldhelpgc = c.nextFree(sizeclass)
			}
			// 走到这儿还是会获取有效的内存块
			x = unsafe.Pointer(v)
			// 内存块清零
			if needzero && span.needzero != 0 {
				memclr(unsafe.Pointer(v), size)
			}
		}
	} else {
		// 前面是小对象（< 32KB）的分配，这里处理大对象的内存分配
		var s *mspan
		shouldhelpgc = true
		systemstack(func() {
			s = largeAlloc(size, needzero)
		})
		s.freeindex = 1
		s.allocCount = 1
		x = unsafe.Pointer(s.base())
		size = s.elemsize
	}

	var scanSize uintptr
	if noscan {
		heapBitsSetTypeNoScan(uintptr(x))
	} else {
		// If allocating a defer+arg block, now that we've picked a malloc size
		// large enough to hold everything, cut the "asked for" size down to
		// just the defer header, so that the GC bitmap will record the arg block
		// as containing nothing at all (as if it were unused space at the end of
		// a malloc block caused by size rounding).
		// The defer arg areas are scanned as part of scanstack.
		if typ == deferType {
			dataSize = unsafe.Sizeof(_defer{})
		}
		heapBitsSetType(uintptr(x), size, dataSize, typ)
		if dataSize > typ.size {
			// Array allocation. If there are any
			// pointers, GC has to scan to the last
			// element.
			if typ.ptrdata != 0 {
				scanSize = dataSize - typ.size + typ.ptrdata
			}
		} else {
			scanSize = typ.ptrdata
		}
		c.local_scan += scanSize
	}

	// Ensure that the stores above that initialize x to
	// type-safe memory and set the heap bits occur before
	// the caller can make x observable to the garbage
	// collector. Otherwise, on weakly ordered machines,
	// the garbage collector could follow a pointer to x,
	// but see uninitialized memory or stale heap bits.
	publicationBarrier()

	// Allocate black during GC.
	// All slots hold nil so no scanning is needed.
	// This may be racing with GC so do it atomically if there can be
	// a race marking the bit.
	if gcphase != _GCoff {
		gcmarknewobject(uintptr(x), size, scanSize)
	}

	if raceenabled {
		racemalloc(x, size)
	}

	if msanenabled {
		msanmalloc(x, size)
	}

	mp.mallocing = 0
	releasem(mp)

	if debug.allocfreetrace != 0 {
		tracealloc(x, size, typ)
	}

	if rate := MemProfileRate; rate > 0 {
		if size < uintptr(rate) && int32(size) < c.next_sample {
			c.next_sample -= int32(size)
		} else {
			mp := acquirem()
			profilealloc(mp, x, size)
			releasem(mp)
		}
	}

	if assistG != nil {
		// Account for internal fragmentation in the assist
		// debt now that we know it.
		assistG.gcAssistBytes -= int64(size - dataSize)
	}

	if shouldhelpgc && gcShouldStart(false) {
		gcStart(gcBackgroundMode, false)
	}

	return x
}
```

堆内存分配遵循了以下基本思路：
* 微对象(< 16B)直接从object分配
* 小对象(< 32KB)以标准的规格从cache.alloc[sizeclass].freelist获取
* 大对象(>= 32KB)直接从堆内存上分配一段连续地址，用bitmap来标记分配的地址

```
func largeAlloc(size uintptr, needzero bool) *mspan {
	// print("largeAlloc size=", size, "\n")

	// 一个内存页是4KB，理论上这里的size+_PageSize是衡大于size的，
	// 这里考虑的是内存溢出的情况
	if size+_PageSize < size {
		throw("out of memory")
	}
	// 算出来分配size大小的内存需要多少个内存页面
	npages := size >> _PageShift
	// 这里考虑的是一个余数的处理，5/2 = 2，然后对2+1最为最终分配的页面数
	if size&_PageMask != 0 {
		npages++
	}

	// 应该是给GC用的，暂时先不看
	deductSweepCredit(npages*_PageSize, npages)

	// 从对上直接分配内存页，这个过程是要加锁的
	s := mheap_.alloc(npages, 0, true, needzero)
	if s == nil {
		throw("out of memory")
	}
	// 初始化bitmap
	s.limit = s.base() + size
	heapBitsForSpan(s.base()).initSpan(s)
	return s
}
```

实际分配heap内存调用了mheap.alloc，看一下它的实现：

```
func (h *mheap) alloc(npage uintptr, sizeclass int32, large bool, needzero bool) *mspan {
	// span抽象了标准规格内存的链表结构
	var s *mspan
	// 这里向systemstack传入一个func调用是将func的上下文切换为g0
	systemstack(func() {
    // 具体实现以后再看
		s = h.alloc_m(npage, sizeclass, large)
	})

	if s != nil {
		// needzero是一个表征是否已经清零过一次，不会重复清零
		if needzero && s.needzero != 0 {
			// 从span的首地址开始，对n个内存页清零
			memclr(unsafe.Pointer(s.base()), s.npages<<_PageShift)
		}
		s.needzero = 0
	}
	return s
}
```

**内存回收实现**

初始化:

```
func gcenable() {
	c := make(chan int, 1)
	// bgsweep里面是一个无限循环，这里会直接往下走
	go bgsweep(c)
	<-c
	memstats.enablegc = true // now that runtime is initialized, GC is okay
}
```

再看bgsweep的实现:

```
func bgsweep(c chan int) {
	sweep.g = getg()

	lock(&sweep.lock)
	sweep.parked = true
	c <- 1
	goparkunlock(&sweep.lock, "GC sweep wait", traceEvGoBlock, 1)

	for {
		// 始终再后台变脸所有的g对象
		for gosweepone() != ^uintptr(0) {
			sweep.nbgsweep++
			Gosched()
		}
		lock(&sweep.lock)
		if !gosweepdone() {
			// 上一次sweep没完成的话，忽略当前的sweep操作
			unlock(&sweep.lock)
			continue
		}
		sweep.parked = true
		goparkunlock(&sweep.lock, "GC sweep wait", traceEvGoBlock, 1)
	}
}
```

gosweepone的实现：

```
func gosweepone() uintptr {
	var ret uintptr
	systemstack(func() {
    // ret => 回收的内存页数
		ret = sweepone()
	})
	return ret
}
```

sweepone的实现：

```
func sweepone() uintptr {
	_g_ := getg()

	// increment locks to ensure that the goroutine is not preempted
	// in the middle of sweep thus leaving the span in an inconsistent state for next GC
	_g_.m.locks++
	sg := mheap_.sweepgen
	// 遍历span链表
	for {
		idx := atomic.Xadd(&sweep.spanidx, 1) - 1
		// span为空链表，返回0
		if idx >= uint32(len(work.spans)) {
			mheap_.sweepdone = 1
			_g_.m.locks--
			if debug.gcpacertrace > 0 && idx == uint32(len(work.spans)) {
				print("pacer: sweep done at heap size ", memstats.heap_live>>20, "MB; allocated ", mheap_.spanBytesAlloc>>20, "MB of spans; swept ", mheap_.pagesSwept, " pages\n")
			}
			return ^uintptr(0)
		}
		s := work.spans[idx]
		// 忽略闲置的span
		if s.state != mSpanInUse {
			// 闲置的span.sweepgen设置为全局sweepgen
			s.sweepgen = sg
			continue
		}
		// atomic.Cas(&s.sweepgen, sg-2, sg-1) => 汇编实现
		// 结果：s.sweepgen == sg-1 ? sg-1 : sg-2
		if s.sweepgen != sg-2 || !atomic.Cas(&s.sweepgen, sg-2, sg-1) {
			continue
		}
		// span.npages => span中的页面数
		// 可见span要么全部被归还heap，要么就全部保留给当前的goroutine
		npages := s.npages
		// span.sweep => 只有2种结果，要么span被全部回收，要么不回收
		if !s.sweep(false) {
			npages = 0
		}
		_g_.m.locks--
		return npages
	}
}
```

span.sweep的实现：

```
func (s *mspan) sweep(preserve bool) bool {
	// It's critical that we enter this function with preemption disabled,
	// GC must not start while we are in the middle of this function.
	// getg应该是从一个G列表里面遍历，具体行为还不清楚
	_g_ := getg()
	if _g_.m.locks == 0 && _g_.m.mallocing == 0 && _g_ != _g_.m.g0 {
		throw("MSpan_Sweep: m is not locked")
	}
	sweepgen := mheap_.sweepgen
	if s.state != mSpanInUse || s.sweepgen != sweepgen-1 {
		print("MSpan_Sweep: state=", s.state, " sweepgen=", s.sweepgen, " mheap.sweepgen=", sweepgen, "\n")
		throw("MSpan_Sweep: bad span state")
	}

	if trace.enabled {
		traceGCSweepStart()
	}

	// 原子操作：mheap.pagesSwept += s.npages
	atomic.Xadd64(&mheap_.pagesSwept, int64(s.npages))

	// span里内存块的sizeclass
	cl := s.sizeclass
	// 避免多次计算sizeclass对应的内存单元大小
	size := s.elemsize
	res := false
	nfree := 0

	var head, end gclinkptr

	// 获取goroutine的cache
	c := _g_.m.mcache
	freeToHeap := false

	// Mark any free objects in this span so we don't collect them.
	// sstart = span的内存开始地址
	sstart := uintptr(s.start << _PageShift)
	// 遍历span的freelist => object链表
	for link := s.freelist; link.ptr() != nil; link = link.ptr().next {
		// 如果节点的地址有问题(检查地址的首尾)会报错
		if uintptr(link) < sstart || s.limit <= uintptr(link) {
			// Free list is corrupted.
			// 收集并打印错误信息
			dumpFreeList(s)
			throw("free list corrupted")
		}
		// heapBitsForAddr返回一个heapBits{bitp, shift}对象
		// setMarkedNonAtomic为不回收的object加标记
		heapBitsForAddr(uintptr(link)).setMarkedNonAtomic()
	}

	// Unlink & free special records for any objects we're about to free.
	// Two complications here:
	// 1. An object can have both finalizer and profile special records.
	//    In such case we need to queue finalizer for execution,
	//    mark the object as live and preserve the profile special.
	// 2. A tiny object can have several finalizers setup for different offsets.
	//    If such object is not marked, we need to queue all finalizers at once.
	// Both 1 and 2 are possible at the same time.
	// 处理一些SetFinalizer的情况，比较复杂，以后细看:
	// http://blog.csdn.net/wang_xijue/article/details/52013262
	specialp := &s.specials
	special := *specialp
	for special != nil {
		// A finalizer can be set for an inner byte of an object, find object beginning.
		p := uintptr(s.start<<_PageShift) + uintptr(special.offset)/size*size
		hbits := heapBitsForAddr(p)
		if !hbits.isMarked() {
			// This object is not marked and has at least one special record.
			// Pass 1: see if it has at least one finalizer.
			hasFin := false
			endOffset := p - uintptr(s.start<<_PageShift) + size
			for tmp := special; tmp != nil && uintptr(tmp.offset) < endOffset; tmp = tmp.next {
				if tmp.kind == _KindSpecialFinalizer {
					// Stop freeing of object if it has a finalizer.
					hbits.setMarkedNonAtomic()
					hasFin = true
					break
				}
			}
			// Pass 2: queue all finalizers _or_ handle profile record.
			for special != nil && uintptr(special.offset) < endOffset {
				// Find the exact byte for which the special was setup
				// (as opposed to object beginning).
				p := uintptr(s.start<<_PageShift) + uintptr(special.offset)
				if special.kind == _KindSpecialFinalizer || !hasFin {
					// Splice out special record.
					y := special
					special = special.next
					*specialp = special
					freespecial(y, unsafe.Pointer(p), size)
				} else {
					// This is profile record, but the object has finalizers (so kept alive).
					// Keep special record.
					specialp = &special.next
					special = *specialp
				}
			}
		} else {
			// object is still live: keep special record
			specialp = &special.next
			special = *specialp
		}
	}

	// Sweep through n objects of given size starting at p.
	// This thread owns the span now, so it can manipulate
	// the block bitmap without atomic operations.

	// layout返回3个值
	// size: elemsize => 对应sizeclass的内存单元大小
	// n: total / size
	// total: s.npages << _PageShift
	size, n, _ := s.layout()
	// 遍历整个span，收集不可达和未标记的object
	// 这里传入的func，会在heapBitsSweepSpan中判断span可以被回收的时候调用
	heapBitsSweepSpan(s.base(), size, n, func(p uintptr) {
		// At this point we know that we are looking at garbage object
		// that needs to be collected.
		if debug.allocfreetrace != 0 {
			tracefree(unsafe.Pointer(p), size)
		}
		if msanenabled {
			msanfree(unsafe.Pointer(p), size)
		}

		// Reset to allocated+noscan.
		// 如果sizeclass == 0 => large span
		if cl == 0 {
			// Free large span.
			// preserve的意思是，内存被goroutine独占，不还给heap
			// 大span不允许preserve
			if preserve {
				throw("can't preserve large span")
			}
			heapBitsForSpan(p).initSpan(s.layout())
			s.needzero = 1

			// Free the span after heapBitsSweepSpan
			// returns, since it's not done with the span.
			freeToHeap = true
		} else {
			// Free small object.
			// 处理小对象span的GC
			if size > 2*sys.PtrSize { // size(object) > 16B, 1.6.2之后的小对象object都是16B
				// >16B object，再前16B加标记
				*(*uintptr)(unsafe.Pointer(p + sys.PtrSize)) = uintptrMask & 0xdeaddeaddeaddead // mark as "needs to be zeroed"
			} else if size > sys.PtrSize { // size(object) == 16B
				// == 16B object, 全部都是标记
				*(*uintptr)(unsafe.Pointer(p + sys.PtrSize)) = 0
			}
			// head, end构建链表，收集不可达的object
			// 注意这里的函数是个回调函数，被调用的时候都是某个对象free的情况
			if head.ptr() == nil {
				head = gclinkptr(p)
			} else {
				end.ptr().next = gclinkptr(p)
			}
			// 新收集到的free object被添加到队尾
			end = gclinkptr(p)
			end.ptr().next = gclinkptr(0x0bade5)
			// 小对象回收计数器+1
			nfree++
		}
	})

	// We need to set s.sweepgen = h.sweepgen only when all blocks are swept,
	// because of the potential for a concurrent free/SetFinalizer.
	// But we need to set it before we make the span available for allocation
	// (return it to heap or mcentral), because allocation code assumes that a
	// span is already swept if available for allocation.
	if freeToHeap || nfree == 0 {
		// The span must be in our exclusive ownership until we update sweepgen,
		// check for potential races.
		// span在被GC扫描的时候不能被使用
		if s.state != mSpanInUse || s.sweepgen != sweepgen-1 {
			print("MSpan_Sweep: state=", s.state, " sweepgen=", s.sweepgen, " mheap.sweepgen=", sweepgen, "\n")
			throw("MSpan_Sweep: bad span state after sweep")
		}
		// 原子操作: span.sweepgen = sweepgen
		atomic.Store(&s.sweepgen, sweepgen)
	}
	if nfree > 0 {
		c.local_nsmallfree[cl] += uintptr(nfree)
		// freeSpan会向heap归还前面检测过来的head->end的空闲object
		// return true if 整个span都被回收，注意对object的回收会是还给central
		res = mheap_.central[cl].mcentral.freeSpan(s, int32(nfree), head, end, preserve)
		// MCentral_FreeSpan updates sweepgen
	} else if freeToHeap {
		// Free large span to heap

		// NOTE(rsc,dvyukov): The original implementation of efence
		// in CL 22060046 used SysFree instead of SysFault, so that
		// the operating system would eventually give the memory
		// back to us again, so that an efence program could run
		// longer without running out of memory. Unfortunately,
		// calling SysFree here without any kind of adjustment of the
		// heap data structures means that when the memory does
		// come back to us, we have the wrong metadata for it, either in
		// the MSpan structures or in the garbage collection bitmap.
		// Using SysFault here means that the program will run out of
		// memory fairly quickly in efence mode, but at least it won't
		// have mysterious crashes due to confused memory reuse.
		// It should be possible to switch back to SysFree if we also
		// implement and then call some kind of MHeap_DeleteSpan.
		if debug.efence > 0 {
			s.limit = 0 // prevent mlookup from finding this span
			sysFault(unsafe.Pointer(uintptr(s.start<<_PageShift)), size)
		} else {
			// 大对象的回收会直接还给heap
			mheap_.freeSpan(s, 1)
		}
		c.local_nlargefree++
		c.local_largefree += size
		res = true
	}
	if trace.enabled {
		traceGCSweepDone()
	}
	return res
}
```
