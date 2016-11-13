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

	stackinit()
	mallocinit()
	mcommoninit(_g_.m)

  // ...
}
```

栈初始化实现：

```
// src/runtime/stack.go
// 全局的span pool, order计算遵循以下公式：
//   order = log_2(size/FixedStack)，其中FixedStack(linux/darwin) = 2kB
// 对Linux/Darwin系统，mSpanList[0..3] -> 2KB, 4KB, 8KB, 16KB
var stackpool [_NumStackOrders]mSpanList

func stackinit() {
	if _StackCacheSize&_PageMask != 0 {
		throw("cache size must be a multiple of page size")
	}
	for i := range stackpool {
    // 初始化小对象内存池
    // 在每个元素上初始化一个双向列表，其中
    // stackpool[0] -> 2KB
    // stackpool[3] -> 16KB
		stackpool[i].init()
	}
	for i := range stackLarge.free {
    // 初始化大对象内存池，也是在每个元素上初始化一个双向链表
		stackLarge.free[i].init()
	}
}
```

分配器初始化实现：

```
// src/runtime/malloc.go

func mallocinit() {
	initSizes()

	if class_to_size[_TinySizeClass] != _TinySize {
		throw("bad TinySizeClass")
	}

	// Check physPageSize.
	if physPageSize == 0 {
		// The OS init code failed to fetch the physical page size.
		throw("failed to get system page size")
	}
	if physPageSize < minPhysPageSize {
		print("system page size (", physPageSize, ") is smaller than minimum page size (", minPhysPageSize, ")\n")
		throw("bad system page size")
	}
	if physPageSize&(physPageSize-1) != 0 {
		print("system page size (", physPageSize, ") must be a power of 2\n")
		throw("bad system page size")
	}

	var p, bitmapSize, spansSize, pSize, limit uintptr
	var reserved bool

	// limit = runtime.memlimit();
	// See https://golang.org/issue/5049
	// TODO(rsc): Fix after 1.1.
	limit = 0

	// Set up the allocation arena, a contiguous area of memory where
	// allocated data will be found. The arena begins with a bitmap large
	// enough to hold 2 bits per allocated word.
	if sys.PtrSize == 8 && (limit == 0 || limit > 1<<30) {
		// On a 64-bit machine, allocate from a single contiguous reservation.
		// 512 GB (MaxMem) should be big enough for now.
		//
		// The code will work with the reservation at any address, but ask
		// SysReserve to use 0x0000XXc000000000 if possible (XX=00...7f).
		// Allocating a 512 GB region takes away 39 bits, and the amd64
		// doesn't let us choose the top 17 bits, so that leaves the 9 bits
		// in the middle of 0x00c0 for us to choose. Choosing 0x00c0 means
		// that the valid memory addresses will begin 0x00c0, 0x00c1, ..., 0x00df.
		// In little-endian, that's c0 00, c1 00, ..., df 00. None of those are valid
		// UTF-8 sequences, and they are otherwise as far away from
		// ff (likely a common byte) as possible. If that fails, we try other 0xXXc0
		// addresses. An earlier attempt to use 0x11f8 caused out of memory errors
		// on OS X during thread allocations.  0x00c0 causes conflicts with
		// AddressSanitizer which reserves all memory up to 0x0100.
		// These choices are both for debuggability and to reduce the
		// odds of a conservative garbage collector (as is still used in gccgo)
		// not collecting memory because some non-pointer block of memory
		// had a bit pattern that matched a memory address.
		//
		// Actually we reserve 544 GB (because the bitmap ends up being 32 GB)
		// but it hardly matters: e0 00 is not valid UTF-8 either.
		//
		// If this fails we fall back to the 32 bit memory mechanism
		//
		// However, on arm64, we ignore all this advice above and slam the
		// allocation at 0x40 << 32 because when using 4k pages with 3-level
		// translation buffers, the user address space is limited to 39 bits
		// On darwin/arm64, the address space is even smaller.
		arenaSize := round(_MaxMem, _PageSize)
		bitmapSize = arenaSize / (sys.PtrSize * 8 / 2)
		spansSize = arenaSize / _PageSize * sys.PtrSize
		spansSize = round(spansSize, _PageSize)
		for i := 0; i <= 0x7f; i++ {
			switch {
			case GOARCH == "arm64" && GOOS == "darwin":
				p = uintptr(i)<<40 | uintptrMask&(0x0013<<28)
			case GOARCH == "arm64":
				p = uintptr(i)<<40 | uintptrMask&(0x0040<<32)
			default:
				p = uintptr(i)<<40 | uintptrMask&(0x00c0<<32)
			}
			pSize = bitmapSize + spansSize + arenaSize + _PageSize
			p = uintptr(sysReserve(unsafe.Pointer(p), pSize, &reserved))
			if p != 0 {
				break
			}
		}
	}

	if p == 0 {
		// On a 32-bit machine, we can't typically get away
		// with a giant virtual address space reservation.
		// Instead we map the memory information bitmap
		// immediately after the data segment, large enough
		// to handle the entire 4GB address space (256 MB),
		// along with a reservation for an initial arena.
		// When that gets used up, we'll start asking the kernel
		// for any memory anywhere.

		// If we fail to allocate, try again with a smaller arena.
		// This is necessary on Android L where we share a process
		// with ART, which reserves virtual memory aggressively.
		// In the worst case, fall back to a 0-sized initial arena,
		// in the hope that subsequent reservations will succeed.
		arenaSizes := []uintptr{
			512 << 20,
			256 << 20,
			128 << 20,
			0,
		}

		for _, arenaSize := range arenaSizes {
			bitmapSize = (_MaxArena32 + 1) / (sys.PtrSize * 8 / 2)
			spansSize = (_MaxArena32 + 1) / _PageSize * sys.PtrSize
			if limit > 0 && arenaSize+bitmapSize+spansSize > limit {
				bitmapSize = (limit / 9) &^ ((1 << _PageShift) - 1)
				arenaSize = bitmapSize * 8
				spansSize = arenaSize / _PageSize * sys.PtrSize
			}
			spansSize = round(spansSize, _PageSize)

			// SysReserve treats the address we ask for, end, as a hint,
			// not as an absolute requirement. If we ask for the end
			// of the data segment but the operating system requires
			// a little more space before we can start allocating, it will
			// give out a slightly higher pointer. Except QEMU, which
			// is buggy, as usual: it won't adjust the pointer upward.
			// So adjust it upward a little bit ourselves: 1/4 MB to get
			// away from the running binary image and then round up
			// to a MB boundary.
			p = round(firstmoduledata.end+(1<<18), 1<<20)
			pSize = bitmapSize + spansSize + arenaSize + _PageSize
			p = uintptr(sysReserve(unsafe.Pointer(p), pSize, &reserved))
			if p != 0 {
				break
			}
		}
		if p == 0 {
			throw("runtime: cannot reserve arena virtual address space")
		}
	}

	// PageSize can be larger than OS definition of page size,
	// so SysReserve can give us a PageSize-unaligned pointer.
	// To overcome this we ask for PageSize more and round up the pointer.
	p1 := round(p, _PageSize)

	mheap_.spans = (**mspan)(unsafe.Pointer(p1))
	mheap_.bitmap = p1 + spansSize + bitmapSize
	if sys.PtrSize == 4 {
		// Set arena_start such that we can accept memory
		// reservations located anywhere in the 4GB virtual space.
		mheap_.arena_start = 0
	} else {
		mheap_.arena_start = p1 + (spansSize + bitmapSize)
	}
	mheap_.arena_end = p + pSize
	mheap_.arena_used = p1 + (spansSize + bitmapSize)
	mheap_.arena_reserved = reserved

	if mheap_.arena_start&(_PageSize-1) != 0 {
		println("bad pagesize", hex(p), hex(p1), hex(spansSize), hex(bitmapSize), hex(_PageSize), "start", hex(mheap_.arena_start))
		throw("misrounded allocation in mallocinit")
	}

	// Initialize the rest of the allocator.
	mheap_.init(spansSize)
	_g_ := getg()
	_g_.m.mcache = allocmcache()
}
```

调度器初始化：

```
// src/runtime/proc.go
func mcommoninit(mp *m) {
	_g_ := getg()

	// g0 stack won't make sense for user (and is not necessary unwindable).
	if _g_ != _g_.m.g0 {
		callers(1, mp.createstack[:])
	}

	mp.fastrand = 0x49f6428a + uint32(mp.id) + uint32(cputicks())
	if mp.fastrand == 0 {
		mp.fastrand = 0x49f6428a
	}

	lock(&sched.lock)
	mp.id = sched.mcount
	sched.mcount++
	checkmcount()
	mpreinit(mp)
	if mp.gsignal != nil {
		mp.gsignal.stackguard1 = mp.gsignal.stack.lo + _StackGuard
	}

	// Add to allm so garbage collector doesn't free g->m
	// when it is just in a register or thread-local storage.
	mp.alllink = allm

	// NumCgoCall() iterates over allm w/o schedlock,
	// so we need to publish it safely.
	atomicstorep(unsafe.Pointer(&allm), unsafe.Pointer(mp))
	unlock(&sched.lock)

	// Allocate memory to hold a cgo traceback if the cgo call crashes.
	if iscgo || GOOS == "solaris" || GOOS == "windows" {
		mp.cgoCallers = new(cgoCallers)
	}
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
