; Generated x86-64 Assembly for Ferret
; Target: Linux x86-64

section .data
pi: dq __float64__(3.141600)
radius: dd 5
x: dd 42
y: dd 10

section .bss
sum: resb 4    ; variable sum (runtime initialized)

section .text
global _start

_start:
    push rbp
    mov rbp, rsp
    ; Variable declaration: x
    ; x already initialized in data section
    ; Variable declaration: y
    ; y already initialized in data section
    ; Variable declaration: sum
    mov rax, [x]
    push rax    ; save left operand
    mov rax, [y]
    mov rbx, rax    ; move right operand to rbx
    pop rax     ; restore left operand
    add rax, rbx
    mov [sum], rax    ; store computed value in sum
    ; Exit program
    mov rax, 60      ; sys_exit
    mov rdi, 0       ; exit status
    syscall

