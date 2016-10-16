"".main t=1 size=64 value=0 args=0x0 locals=0x8
	0x0000 00000 (examples/e3/e3.go:3)	TEXT	"".main(SB), $8-0
	0x0000 00000 (examples/e3/e3.go:3)	MOVQ	(TLS), CX
	0x0009 00009 (examples/e3/e3.go:3)	CMPQ	SP, 16(CX)
	0x000d 00013 (examples/e3/e3.go:3)	JLS	52
	0x000f 00015 (examples/e3/e3.go:3)	SUBQ	$8, SP
	0x0013 00019 (examples/e3/e3.go:3)	FUNCDATA	$0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0013 00019 (examples/e3/e3.go:3)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0013 00019 (examples/e3/e3.go:4)	PCDATA	$0, $0
	0x0013 00019 (examples/e3/e3.go:4)	CALL	runtime.printlock(SB)
	0x0018 00024 (examples/e3/e3.go:4)	MOVQ	$1, (SP)
	0x0020 00032 (examples/e3/e3.go:4)	PCDATA	$0, $0
	0x0020 00032 (examples/e3/e3.go:4)	CALL	runtime.printint(SB)
	0x0025 00037 (examples/e3/e3.go:4)	PCDATA	$0, $0
	0x0025 00037 (examples/e3/e3.go:4)	CALL	runtime.printnl(SB)
	0x002a 00042 (examples/e3/e3.go:4)	PCDATA	$0, $0
	0x002a 00042 (examples/e3/e3.go:4)	CALL	runtime.printunlock(SB)
	0x002f 00047 (examples/e3/e3.go:5)	ADDQ	$8, SP
	0x0033 00051 (examples/e3/e3.go:5)	RET
	0x0034 00052 (examples/e3/e3.go:5)	NOP
	0x0034 00052 (examples/e3/e3.go:3)	CALL	runtime.morestack_noctxt(SB)
	0x0039 00057 (examples/e3/e3.go:3)	JMP	0
	0x0000 65 48 8b 0c 25 00 00 00 00 48 3b 61 10 76 25 48  eH..%....H;a.v%H
	0x0010 83 ec 08 e8 00 00 00 00 48 c7 04 24 01 00 00 00  ........H..$....
	0x0020 e8 00 00 00 00 e8 00 00 00 00 e8 00 00 00 00 48  ...............H
	0x0030 83 c4 08 c3 e8 00 00 00 00 eb c5 cc cc cc cc cc  ................
	rel 5+4 t=14 +0
	rel 20+4 t=6 runtime.printlock+0
	rel 33+4 t=6 runtime.printint+0
	rel 38+4 t=6 runtime.printnl+0
	rel 43+4 t=6 runtime.printunlock+0
	rel 53+4 t=6 runtime.morestack_noctxt+0
"".init t=1 size=80 value=0 args=0x0 locals=0x0
	0x0000 00000 (examples/e3/e3.go:5)	TEXT	"".init(SB), $0-0
	0x0000 00000 (examples/e3/e3.go:5)	MOVQ	(TLS), CX
	0x0009 00009 (examples/e3/e3.go:5)	CMPQ	SP, 16(CX)
	0x000d 00013 (examples/e3/e3.go:5)	JLS	62
	0x000f 00015 (examples/e3/e3.go:5)	NOP
	0x000f 00015 (examples/e3/e3.go:5)	NOP
	0x000f 00015 (examples/e3/e3.go:5)	FUNCDATA	$0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x000f 00015 (examples/e3/e3.go:5)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x000f 00015 (examples/e3/e3.go:5)	MOVBQZX	"".initdone·(SB), BX
	0x0016 00022 (examples/e3/e3.go:5)	CMPB	BL, $0
	0x0019 00025 (examples/e3/e3.go:5)	JEQ	47
	0x001b 00027 (examples/e3/e3.go:5)	MOVBQZX	"".initdone·(SB), BX
	0x0022 00034 (examples/e3/e3.go:5)	CMPB	BL, $2
	0x0025 00037 (examples/e3/e3.go:5)	JNE	40
	0x0027 00039 (examples/e3/e3.go:5)	RET
	0x0028 00040 (examples/e3/e3.go:5)	PCDATA	$0, $0
	0x0028 00040 (examples/e3/e3.go:5)	CALL	runtime.throwinit(SB)
	0x002d 00045 (examples/e3/e3.go:5)	UNDEF
	0x002f 00047 (examples/e3/e3.go:5)	MOVB	$1, "".initdone·(SB)
	0x0036 00054 (examples/e3/e3.go:5)	MOVB	$2, "".initdone·(SB)
	0x003d 00061 (examples/e3/e3.go:5)	RET
	0x003e 00062 (examples/e3/e3.go:5)	NOP
	0x003e 00062 (examples/e3/e3.go:5)	CALL	runtime.morestack_noctxt(SB)
	0x0043 00067 (examples/e3/e3.go:5)	JMP	0
	0x0000 65 48 8b 0c 25 00 00 00 00 48 3b 61 10 76 2f 0f  eH..%....H;a.v/.
	0x0010 b6 1d 00 00 00 00 80 fb 00 74 14 0f b6 1d 00 00  .........t......
	0x0020 00 00 80 fb 02 75 01 c3 e8 00 00 00 00 0f 0b c6  .....u..........
	0x0030 05 00 00 00 00 01 c6 05 00 00 00 00 02 c3 e8 00  ................
	0x0040 00 00 00 eb bb cc cc cc cc cc cc cc cc cc cc cc  ................
	rel 5+4 t=14 +0
	rel 18+4 t=13 "".initdone·+0
	rel 30+4 t=13 "".initdone·+0
	rel 41+4 t=6 runtime.throwinit+0
	rel 49+4 t=13 "".initdone·+-1
	rel 56+4 t=13 "".initdone·+-1
	rel 63+4 t=6 runtime.morestack_noctxt+0
gclocals·33cdeccccebe80329f1fdbee7f5874cb t=8 dupok size=8 value=0
	0x0000 01 00 00 00 00 00 00 00                          ........
gclocals·33cdeccccebe80329f1fdbee7f5874cb t=8 dupok size=8 value=0
	0x0000 01 00 00 00 00 00 00 00                          ........
gclocals·33cdeccccebe80329f1fdbee7f5874cb t=8 dupok size=8 value=0
	0x0000 01 00 00 00 00 00 00 00                          ........
gclocals·33cdeccccebe80329f1fdbee7f5874cb t=8 dupok size=8 value=0
	0x0000 01 00 00 00 00 00 00 00                          ........
"".initdone· t=31 size=1 value=0
"".main·f t=8 dupok size=8 value=0
	0x0000 00 00 00 00 00 00 00 00                          ........
	rel 0+8 t=1 "".main+0
"".init·f t=8 dupok size=8 value=0
	0x0000 00 00 00 00 00 00 00 00                          ........
	rel 0+8 t=1 "".init+0
