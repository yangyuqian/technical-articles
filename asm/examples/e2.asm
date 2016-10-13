SECTION .data

SYS_WRITE: equ 0x2000004
SYS_EXIT: equ 0x2000001
STD_OUT: equ 1
EXIT_CODE: equ 1
NEW_LINE: equ 0xa
WRONG_ARGC: db "Must be two command line argument", 0xa
LEN_WONG_ARGC: equ $-WRONG_ARGC

SECTION .text
global _main

_main:
	;; get argc by pop top of stack to RCX
	pop rcx
	cmp rcx, 3 ;; argc = 3 means there are 2 args
	jne _wrong_argc
	jmp _exit

_wrong_argc:
	mov rsi, WRONG_ARGC
	mov rdx, LEN_WONG_ARGC
	jmp _print

;; print a str to console
;; RSI will keep the msg
;; RDX will keep the len(msg) - bytes
_print:
	mov rax, SYS_WRITE
	mov rdi, STD_OUT
	;; mov rsi, MSG_DATA
	syscall
	jmp _exit

_exit:
	mov rax, SYS_EXIT
	mov rdi, EXIT_CODE
	syscall

