; Generated x86-64 Assembly for Ferret
; Target: Linux x86-64

section .data
pi: dq 3.141590

section .bss
radius: resb 4    ; variable radius (runtime initialized)
area: resb 8    ; variable area (runtime initialized)

section .text
global _start

_start:
    push rbp
    mov rbp, rsp
    ; Import: app/maths/math
    ; Variable declaration: radius
    mov rax, 7
    mov [radius], rax    ; store computed value in radius
    ; Variable declaration: area
    mov rax, [pi]
    push rax    ; save left operand
    mov rax, [radius]
    mov rbx, rax    ; move right operand to rbx
    pop rax     ; restore left operand
    imul rax, rbx
    push rax    ; save left operand
    mov rax, [radius]
    mov rbx, rax    ; move right operand to rbx
    pop rax     ; restore left operand
    imul rax, rbx
    mov [area], rax    ; store computed value in area
    ; Exit program
    mov rax, 60      ; sys_exit
    mov rdi, 0       ; exit status
    syscall

