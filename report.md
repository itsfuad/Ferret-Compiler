If we use file based import,

```rs

import "std/io"; // std/io.fer (extension added automatically)
import "std/math"; // std/math.fer (extension added automatically)

```

If we use module based import,

```rs

import "std/io"; // std/io/*.fer (extension added automatically)
import "std/math"; // std/math/*.fer (extension added automatically)

```

Package: Collection of source files, located in the same directory.
Module: Collection of packages. Folder structure is not important.


----------------
```
.
| main.fer
├── std
│   ├── io.fer
│   ├── math.fer
│   └── fmt.fer
├── mylib
│   └── utils.fer
├── mylib2
│   └── helpers.fer
└── mylib3
    ├── utils.fer
    └── helpers.fer
```
Some external via url:
```
. (github.com/markov/graphics)
| main.fer
├── graphics
│   ├── shapes.fer
│   └── colors.fer
├── network
│   ├── http.fer
│   └── ftp.fer
└── utils
    ├── string.fer
    └── file.fer
```
```rs
import "std/io";
import "std/math";
import "std/fmt";
import "mylib/utils";

// For external libraries
import "github.com/markov/graphics/shapes";
import "github.com/markov/graphics/colors";
import "github.com/markov/network/http";

fn main() {
    let result = math::sqrt(16.0);
    fmt::println("The square root of 16 is: {}", result);
    let file, err = io::open("example.txt", io::READ);
    if err != null {
        fmt::println("Error opening file: {}", err);
        return;
    }

    //create window
    let window, err = graphics::create_window("My Window", 800, 600);
    if err != null {
        fmt::println("Error creating window: {}", err);
        return;
    }
    graphics::set_background_color(window, colors::WHITE);
    graphics::draw_circle(window, 400, 300, 50, colors::RED);
    graphics::show(window);
    file.close();
    fmt::println("File opened successfully and window created.");
}
```