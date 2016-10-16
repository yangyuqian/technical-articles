Golang中的Plan9汇编器
-----------------

Golang自1.4之后实现了`自举`，即用Go实现编译器来编译Go源码，从此
Go与机器之间只隔了一层汇编实现，本文中介绍`Plan9汇编`旨在进一步学习
Go编译做准备. Go中`Plan9`的实现和具体的系统架构无关，实现了汇编的跨平台，
下面以`MACOS` + `Intel i7`为例介绍.

为了区分`Plan9`汇编和`Go Plan9 Assembler`，下面所有的描述都会改为
`Go Assembler`.

`Go Assembler`中的虚拟寄存器：

* FP(Frame pointer): arguments and locals.
* PC(Program counter): jumps and branches.
* SB(Static base pointer): global symbols.
* SP(Stack pointer): top of stack.


```
// 所有用户空间的数据都可以通过FP(局部数据)或SB(全局数据)访问
All user-defined symbols are written as offsets to the pseudo-registers
FP (arguments and locals) and SB (globals).

// SB抽象了内存空间，foo(SB)的意思是用foo来代表内存里面的一个地址（赋值？）
The SB pseudo-register can be thought of as the origin of memory,
so the symbol foo(SB) is the name foo as an address in memory.
// foo(SB)可以用来定义全局的function和数据
This form is used to name global functions and data.
// foo<>(SB)和C里面的static一个效果，声明只在当前源码文件中可见
Adding <> to the name, as in foo<>(SB),
makes the name visible only in the current source file,
like a top-level static declaration in a C file.
// 可以在引用上加上偏移量，如前面说的foo(SB)，
// 如果foo+4(SB)的意思是foo + 4 bytes的地址
Adding an offset to the name refers to that offset from the symbol's address,
so foo+4(SB) is four bytes past the start of foo.

// FP用来访问函数的参数
The FP pseudo-register is a virtual frame pointer used to
refer to function arguments.
// 编译器维护了栈上的参数指针
The compilers maintain a virtual frame pointer and refer to
the arguments on the stack as offsets from that pseudo-register.
// 0(FP)就是function的第一个参数
Thus 0(FP) is the first argument to the function,
// 64位系统上8(FP)就是第二个参数，后面加上偏移量就可以访问更多的参数
8(FP) is the second (on a 64-bit machine), and so on.
// 要访问具体function的参数，需要加上name，比如foo+0(FP)获取foo的第一个参数
// foo+8(FP)获取第二个参数
However, when referring to a function argument this way,
it is necessary to place a name at the beginning,
as in first_arg+0(FP) and second_arg+8(FP).
// 编译器中强制要求必须用name来访问FP
The assembler enforces this convention, rejecting plain 0(FP) and 8(FP).

// SP是栈指针，和NASM中的RSP类似
The SP pseudo-register is a virtual stack pointer used to refer to
frame-local variables and the arguments being prepared for function calls.
// SP指向当前local stack frame的栈顶，使用时foo-8(SP)代表foo的栈第8byte
It points to the top of the local stack frame,
so references should use negative offsets in the range
[−framesize, 0): x-8(SP), y-4(SP), and so on.]

// 如果硬件支持SP寄存器，那么不加name的时候就是访问硬件寄存器.
// 因此 x-8(SP)和-8(SP)访问的会是不同的内存空间
On architectures with a hardware register named SP,
the name prefix distinguishes references to the virtual stack pointer
from references to the architectural SP register.
That is, x-8(SP) and -8(SP) are different memory locations:
the first refers to the virtual stack pointer pseudo-register,
while the second refers to the hardware's SP register.
// 对SP和PC的访问都应该带上name，如果要访问对应的硬件寄存器，
// 可以用RSP
On machines where SP and PC are traditionally aliases for a physical,
numbered register, in the Go assembler the names SP and PC are still
treated specially; for instance, references to SP require a symbol,
much like FP. To access the actual hardware register use the true R name.
For example, on the ARM architecture the hardware
SP and PC are accessible as R13 and R15.
```


# References

[A Manual for the Plan 9 assembler](http://plan9.bell-labs.com/sys/doc/asm.html)

[A Quick Guide to Go's Assembler](https://golang.org/doc/asm)
