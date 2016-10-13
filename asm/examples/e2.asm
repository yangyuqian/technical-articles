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
	;; it can be pop to any 64 bit register, i.e
	;; pop rax
	pop rcx
	;; if argc != 3, jmp to _wrong_argc
	;; argc = 3 means there are 2 args
	cmp rcx, 3
	jne _wrong_argc
	;; get arg[0]
	add rsp, 8
	pop rsi
	;; args[0].to_numbder
	call _str_to_num
	;; number stored in rax
	mov r10, rax
	;; args[1].to_numbder
	pop rsi
	call _str_to_num
  mov r11, rax
	;; args[0] + args[1]
	add r10, r11
	;; rax = r10
	mov rax, r10


	call _exit

;; transform str to number
_str_to_num:
  ;; set rax to 0
	xor rax, rax
	mov rcx, 10

_next:
	cmp [rsi], byte 0
	je _return
	mov bl, [rsi]
	;; num = ascii - 48
	sub bl, 48
	;; rbx = rcx * bl
	mul rcx
	;; rax = rax + rbx
	add rax, rbx
	;; rsi ++
	inc rsi
	jmp _next

_return:
	ret

;; sum result is saved in rax
_num_to_str:
	xor r12, r12
	mov rdx, 0
	mov rbx, 10
	div rbx
	;; ascii = num + 48
	add rdx, 48
	add rdx, 0x0
	;; push character to stack
	push rdx
	cmp rax, 0x0
	jne _num_to_str
	;; TODO: print the result
	;; TODO

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

