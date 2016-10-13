汇编开发
--------------

本文基于MAC介绍一些基本汇编开发，旨在帮助理解程序的底层执行原理.
具体的系统架构和操作系统都不重要，不同系统架构带来的无非是指令集和寄存器等差异，
最终程序的本质都是一样的.

字节定义

```
a byte as 8 bits, a word as 16 bits,
a double word as 32 bits, a quadword as 64 bits,
and a double quadword as 128 bits.
```

Intel数据存储方式

```
Intel stores bytes "little endian,"
meaning lower significant bytes are stored in lower memory addresses.
```

Little-endian - we can imagine memory as one large array. It contains bytes.
Each address stores one element of the memory “array”.
Each element is one byte. For example we have 4 bytes: AA 56 AB FF.
In little-endian the least significant byte has the smallest address:

```
0 FF
1 AB
2 56
3 AA
```

**Intel i7寄存器**

1. 64 bits

RAX, RBX, RCX, RDX, RBP, RSI, RDI, RSP, R8~R15. 共16个64bits寄存器;

对于前面8个：RAX, RBX, RCX, RDX, RBP, RSI, RDI, RSP

如果R改成E，比如RAX - EAX，就可以访问低32bits;
还可以把前面的R去掉，比如RAX - AX，访问低16bits; AL访问RAX的低8bits;
AX的高8bits可以用AH访问;

对于后面8个新寄存器：R8 - R15

如 R8, R8 (64bits), R8D (低32bits), R8W (低16bits), R8B (低8bits)

还有一个64bits的寄存器RIP，充当PC（程序计数器），存放下一条指令的地址

还有RSP存放栈顶; RFLAGS存放判断结果.

2. FPU(floating pointing unit)

The floating point unit (FPU) contains eight registers FPR0-FPR7

Single Instruction Multiple Data (SIMD) instructions execute a single command
on multiple pieces of data in parallel and are a common usage for assembly
routines. MMX and SSE commands (using the MMX and XMM registers respectively)
support SIMD operations, which perform an instruction on up to eight pieces of
data in parallel.
For example, eight bytes can be added to eight bytes in one instruction using MMX.

**NASM汇编**

本文基于MACOS 10.10，汇编器用`nasm`:

```
$ nasm -v
NASM version 2.12.01 compiled on Mar 23 2016
```

一定要保证`nasm`用比较新的版本，老版本可能不支持64bits汇编的编译.

本文中所有的例子都按照`e${order}`的格式编号，比如例1，文件就是`examples/e1.asm`.

通过如下命令来编译，检查编译环境是否满足要求：

```
// 生成Object文件
$ nasm -f macho64 -o play/e1.o examples/e1.asm
// 生成可执行文件，这里加上-e _main是因为nasm默认的入口函数的_start
$ ld -o play/e1 -e _main play/e1.o
// 运行可执行文件
$ ./play/e1

Hello, World!
```

也可以通过`build.sh`来编译和运行例子：

```
# 编译但不运行e1
$ sh build.sh e1 # e1也可以是e1.asm

Executable of e1 is located at /Users/yangyuqian/code/technical-articles/asm/play/e1

# 编译并运行e1
$ sh build.sh e1 exec

=============== Run example e1 ==============

+ /Users/yangyuqian/code/technical-articles/asm/play/e1
Hello, World!
```

# Hello World

例1: `examples/e1.asm` 是`Hello World`程序，向console输出一段字符

每个汇编程序都可以有3个`section`:

* `data`: 数据段, 用来存放常量
* `text`: 代码段
* `bss`: 用来存放程序中未初始化的全局变量的一块内存区域

首先看`data section`:

```
// 声明下面的代码都在数据段中
SECTION .data
// 定义变量
msg: db "Hello, World!", 0x0a
len: equ $-msg
```

说到常量的定义：

```
SECTION .data

// const1 = 100
const1: equ 100
```

那么上面的`db` `equ`是什么意思呢？这是`nasm`支持的 [pseudo-instructions](http://www.nasm.us/doc/nasmdoc3.html).

```
db    0x55                ; just the byte 0x55
db    0x55,0x56,0x57      ; three bytes in succession
db    'hello',13,10,'$'   ; so are string constants
dw    0x1234              ; 0x34 0x12
dw    'ab'                ; 0x61 0x62 (character constant)
```

所以上面的 `msg: db "xxx", 0x0a` 就是简单的定义了一个字符串常量.

用`$-data`可以获取`data`的数据长度，所以`len: equ $-msg`就定义了`len=len(msg)`.

```
message         db      'hello, world'
msglen          equ     $-message
```

还可以定义没有初始化的常量:

```
buffer:         resb    64              ; reserve 64 bytes
wordvar:        resw    1               ; reserve a word
realarray       resq    10              ; array of ten reals
ymmval:         resy    1               ; one YMM register
zmmvals:        resz    32              ; 32 ZMM registers
```

然后看`text section`, 正式进入代码逻辑：

```
SECTION .text
// 声明入口是_main
global _main

// 定义一个函数
kernel:
    syscall
    ret

// 程序入口函数
_main:
    // 表明当前syscall是写I/O操作
    mov rax,0x2000004
    // 表明当前写stdout
    mov rdi,1
    // 数据入口地址写入rsi
    mov rsi,msg
    // 数据长度写入rdx
    mov rdx,len
    call kernel

    // 表明当前系统调用是exit
    mov rax,0x2000001
    // exit的时候返回0，等价于`exit 0`
    mov rdi,0
    call kernel
```

在MAC中系统调用的标志比较有意思，需要把具体的值加上`0x2000000`.

例2 `examples/e2.asm`是一个更复杂一点的程序，从命令行获取2个参数，
判断参数相加只和是否是10，输出一段结果.




# References

[Intel i7 Assembly](https://software.intel.com/en-us/articles/introduction-to-x64-assembly)

[Say hello to x64 Assembly](http://0xax.blogspot.ca/2014/08/say-hello-to-x64-assembly-part-1.html)

[Examples](https://github.com/0xAX/asm)

[NASM Assembly](http://www.nasm.us/doc/nasmdoc3.html)

[Making system calls from Assembly in Mac OS X](https://filippo.io/making-system-calls-from-assembly-in-mac-os-x/)
