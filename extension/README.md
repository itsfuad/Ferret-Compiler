# Ferret Language Support

VS Code extension providing syntax highlighting and basic language features for the [Ferret programming language](https://github.com/itsfuad/Ferret-Compiler).

## Features

* Syntax highlighting for `.fer` source files
* Syntax highlighting for `.fer.ret` (project configuration) files
* Basic language server support for error checking and autocomplete (if Ferret compiler is available)

## Installation

1. Download the latest `.vsix` file from the [Releases](https://github.com/itsfuad/Ferret-Language-Support/releases).
2. Install it in VS Code:

   ```bash
   code --install-extension ferret-x.y.z-*.vsix
   ```
3. Restart VS Code.

## Usage

* Open any `.fer` file to get syntax highlighting.

## Requirements

* (Optional) [Ferret Compiler](https://github.com/itsfuad/Ferret-Compiler) available in `PATH` for advanced features.

## License

[MIT](LICENSE)