// Multiple assignment
let a, b, c := 1, 2, 4;

// Arrays
let arr: [10]i32 = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];  // Fixed-size
let dynArr: []i32 = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9];  // Dynamic-size

// Functions
fn getPi() -> f32 {
    return 3.14;
}

// Compile-time constant
const pi := comptime getPi();

// enum
type Color enum {
    Red,
    Green,
    Blue
};

// Structs
type Circle struct {
    .radius: f32,
    .area: f32,
    .color: Color
};

// Interfaces
type Shape interface {
    fn area(self) -> f32,
    fn perimeter(self) -> f32
};


// Methods
fn (c &Circle) area() -> f32 {
    return c.area;
}
fn (c &Circle) perimeter() -> f32 {
    return 2.0 * pi * c.radius;
}

// Struct literal
const circle1 := Circle {
    .radius = 5.0,
    .area = pi * 5.0 * 5.0,
    .color = Color::Red
};

// Result type & error handling
fn fetch(url: str) -> Result ! str {
    if url == "https://example.com/api/data" {
        return Result{.data = "Fetched Data"};
    } else {
        return "Network Error"!;
    }
}

// Catch with default value
let res := fetch("https://example.com/api/data") catch err {
    log("Error fetching data: " + err);
    with "default data";
};

// Loops
for i, v in 0..10:2 {
    log("Current index: " + i);
    log("Current value: " + v);
}

while a < 10 {
    a += 1;
}

// Optional types
let mayExist: i32?;

// Statement-only `when`
when mayExist {
    null => log("mayExist is null"),
    _ => log("mayExist has a value: " + mayExist)
}

// Optional unwrapping in `if`
if mayExist == null {
    log("mayExist is null");
} else {
    log("mayExist has a value: " + mayExist);
}

// Ternary / Elvis operators
let value  := mayExist ? mayExist : 42;  // Standard ternary
let value2 := mayExist ?: 42;            // Elvis operator
