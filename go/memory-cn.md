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
