; Generated x86-64 Assembly for Ferret
; Target: Linux x86-64

section .data

str_1: db 'Hello ', 0
str_2: db 'World', 0
str_3: db 'John', 0
str_4: db 'This value is declared later in the code.', 0
section .bss
x: resb 4    ; variable x (runtime initialized)
y: resb 8    ; variable y (runtime initialized)
z: resb 8    ; variable z (runtime initialized)
str1: resb 8    ; variable str1 (runtime initialized)
str2: resb 8    ; variable str2 (runtime initialized)
greeting: resb 8    ; variable greeting (runtime initialized)
a: resb 4    ; variable a (runtime initialized)
b: resb 4    ; variable b (runtime initialized)
sum: resb 4    ; variable sum (runtime initialized)
name: resb 8    ; variable name (runtime initialized)
age: resb 4    ; variable age (runtime initialized)
importedData: resb 8    ; variable importedData (runtime initialized)
pi: resb 8    ; variable pi (runtime initialized)
largeNumber: resb 4    ; variable largeNumber (runtime initialized)
octal: resb 4    ; variable octal (runtime initialized)
hex: resb 4    ; variable hex (runtime initialized)
binary: resb 4    ; variable binary (runtime initialized)
scientific: resb 8    ; variable scientific (runtime initialized)
cxx: resb 8    ; variable cxx (runtime initialized)
laterDeclaredValue: resb 8    ; variable laterDeclaredValue (runtime initialized)

section .text
global _start

_start:
    push rbp
    mov rbp, rsp
    ; Import: app/data
    ; Variable declaration: x
    mov rax, 42
    mov [x], rax    ; store computed value in x
    ; Variable declaration: y
    mov rax, 100
    mov [y], rax    ; store computed value in y
    ; Variable declaration: z
    mov rax, [x]
    push rax    ; save left operand
    mov rax, [y]
    mov rbx, rax    ; move right operand to rbx
    pop rax     ; restore left operand
    add rax, rbx
    mov [z], rax    ; store computed value in z
    ; Variable declaration: str1
    mov rax, str_1
    mov [str1], rax    ; store computed value in str1
    ; Variable declaration: str2
    mov rax, str_2
    mov [str2], rax    ; store computed value in str2
    ; Variable declaration: greeting
    mov rax, [str1]
    push rax    ; save left operand
    mov rax, [str2]
    mov rbx, rax    ; move right operand to rbx
    pop rax     ; restore left operand
    add rax, rbx
    mov [greeting], rax    ; store computed value in greeting
    ; Variable declaration: a
    mov rax, 5
    mov [a], rax    ; store computed value in a
    ; Variable declaration: b
    mov rax, 10
    mov [b], rax    ; store computed value in b
    ; Variable declaration: sum
    mov rax, 1
    push rax    ; save left operand
    mov rax, 3
    mov rbx, rax    ; move right operand to rbx
    pop rax     ; restore left operand
    add rax, rbx
    mov [sum], rax    ; store computed value in sum
    ; Variable declaration: name
    mov rax, str_3
    mov [name], rax    ; store computed value in name
    ; Variable declaration: age
    mov rax, 30
    mov [age], rax    ; store computed value in age
    ; Variable declaration: importedData
    mov rax, [myData]
    mov [importedData], rax    ; store computed value in importedData
    ; Variable declaration: pi
    mov rax, 3    ; float 3.141590 as int
    mov [pi], rax    ; store computed value in pi
    ; Variable declaration: largeNumber
    mov rax, 1000000
    mov [largeNumber], rax    ; store computed value in largeNumber
    ; Variable declaration: octal
    mov rax, 493
    mov [octal], rax    ; store computed value in octal
    ; Variable declaration: hex
    mov rax, 255
    mov [hex], rax    ; store computed value in hex
    ; Variable declaration: binary
    mov rax, 42
    mov [binary], rax    ; store computed value in binary
    ; Variable declaration: scientific
    mov rax, 12300    ; float 12300.000000 as int
    mov [scientific], rax    ; store computed value in scientific
    ; Variable declaration: cxx
    mov rax, [kk]
    mov [cxx], rax    ; store computed value in cxx
    ; Variable declaration: laterDeclaredValue
    mov rax, str_4
    mov [laterDeclaredValue], rax    ; store computed value in laterDeclaredValue
    ; Exit program
    mov rax, 60      ; sys_exit
    mov rdi, 0       ; exit status
    syscall

