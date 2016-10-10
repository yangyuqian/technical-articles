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

Intel i7寄存器

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

本文基于MACOS 10.10，汇编器用`nasm`:

```
$ nasm -v
NASM version 2.12.01 compiled on Mar 23 2016
```

一定要保证`nasm`用比较新的版本，老版本可能不支持64bits汇编的编译.

# 实例介绍

例1: `examples/e1.asm` 是`Hello World`程序，向console输出一段字符，
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




# References

[Intel i7 Assembly](https://software.intel.com/en-us/articles/introduction-to-x64-assembly)

[Say hello to x64 Assembly](http://0xax.blogspot.ca/2014/08/say-hello-to-x64-assembly-part-1.html)

[Examples](https://github.com/0xAX/asm)
