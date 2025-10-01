package main

import (
	"fmt"
	"encoding/json"
)

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *LspError   `json:"error,omitempty"`
}

type LspError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	fmt.Println("LSP server main function")
}