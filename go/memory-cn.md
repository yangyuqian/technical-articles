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

