package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"compiler/cmd"
	"compiler/config"
	"compiler/report"
	"lsp/wio"
)

// Track files that have had diagnostics published
var filesWithDiagnostics = make(map[string]bool)

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
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// CLI flag for port
	portFlag := flag.Int("port", 0, "Port to listen on (0 = choose a free port dynamically)")
	flag.Parse()

	listenAddr := fmt.Sprintf("127.0.0.1:%d", *portFlag)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to start server on %s: %v", listenAddr, err)
	}
	defer listener.Close()

	// Get actual port (dynamic or fixed)
	port := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("PORT:%d\n", port) // For VS Code extension auto-connect
	os.Stdout.Sync()

	log.Printf("LSP Server listening on %s", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		log.Printf("Client connected: %s", conn.RemoteAddr())
		go func(c net.Conn) {
			defer c.Close()
			handleConnection(c)
			log.Printf("Client disconnected: %s", c.RemoteAddr())
		}(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleConnection: %v", r)
		}
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		msg, err := readMessage(reader)
		if err == io.EOF {
			log.Printf("Client disconnected")
			break
		}
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		if msg == "" {
			log.Printf("Empty message received, skipping")
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(msg), &req); err != nil {
			log.Printf("Invalid JSON message %q: %v", msg, err)
			continue
		}

		switch req.Method {
		case "initialize":
			handleInitialize(writer, req)
		case "textDocument/didOpen", "textDocument/didChange", "textDocument/didSave":
			handleTextDocumentChange(writer, req)
		case "shutdown":
			handleShutdown(writer, req)
		case "exit":
			handleExit(writer, conn)
		default:
			handleUnknownMethod(req)
		}
	}
}

func handleInitialize(writer *bufio.Writer, req Request) {
	resp := Response{
		Jsonrpc: "2.0",
		Id:      req.Id,
		Result: map[string]interface{}{
			"capabilities": map[string]interface{}{
				"textDocumentSync": 1,
			},
		},
	}
	writeMessage(writer, resp)

	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]interface{}{},
	}
	writeRawMessage(writer, notification)
}

func handleTextDocumentChange(writer *bufio.Writer, req Request) {
	var params map[string]interface{}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Printf("Invalid params: %v", err)
		return
	}

	uri, ok := params["textDocument"].(map[string]interface{})["uri"].(string)
	if !ok {
		log.Printf("Invalid uri: %v", params)
		return
	}

	processDiagnostics(writer, uri)
}

func handleExit(writer *bufio.Writer, conn net.Conn) {
	log.Printf("Client requested exit")

	// flush pending messages
	if writer != nil {
		writer.Flush()
	}

	// defer closing conn until after graceful shutdown
	if conn != nil {
		conn.Close()
	}

	// exit cleanly
	os.Exit(0)
}

func handleShutdown(writer *bufio.Writer, req Request) {
	resp := Response{
		Jsonrpc: "2.0",
		Id:      req.Id,
		Result:  nil,
	}
	writeMessage(writer, resp)

	log.Println("Server received shutdown request")
}

func handleUnknownMethod(req Request) {
	log.Printf("Unknown method: %v", req.Method)
}

func readMessage(reader *bufio.Reader) (string, error) {
	contentLength := 0

	// Read headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		// Trim both \r and \n
		line = strings.TrimRight(line, "\r\n")

		if line == "" { // End of headers
			break
		}

		if strings.HasPrefix(line, "Content-Length: ") {
			lengthStr := strings.TrimPrefix(line, "Content-Length: ")
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return "", fmt.Errorf("invalid Content-Length: %v", err)
			}
			log.Printf("Content length: %d", contentLength)
		}
	}

	if contentLength == 0 {
		return "", fmt.Errorf("no content length header found")
	}

	// Read body
	body := make([]byte, contentLength)
	n, err := io.ReadFull(reader, body)
	if err != nil {
		return "", fmt.Errorf("failed to read message body (read %d of %d bytes): %v", n, contentLength, err)
	}

	bodyStr := string(body)
	log.Printf("Received message body: %q", bodyStr)
	return bodyStr, nil
}

func writeMessage(writer *bufio.Writer, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	if _, err := writer.WriteString(msg); err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("Failed to flush response: %v", err)
	}
}

func writeRawMessage(writer *bufio.Writer, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	fullMsg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	if _, err := writer.WriteString(fullMsg); err != nil {
		log.Printf("Failed to write message: %v", err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("Failed to flush message: %v", err)
	}
}

func processDiagnostics(writer *bufio.Writer, uri string) {

	var reports report.Reports

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in processDiagnostics: %v", r)
			makeDiagnostics(nil, writer, uri)
		}
	}()

	log.Println("Processing diagnostics for:", uri)

	filePath, err := wio.UriToFilePath(uri)
	if err != nil {
		log.Println("Error converting URI to file path:", err)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	log.Println("File path:", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", filePath)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	projectRoot, err := config.GetProjectRoot(filePath)
	if err != nil {
		log.Println("Error finding project root:", err)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	conf, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		log.Println("Error loading project configuration:", err)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	context := cmd.Compile(conf, false)

	// Check if context or Reports is nil
	if context == nil {
		log.Println("Compilation context is nil")
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	reports = context.Reports

	makeDiagnostics(reports, writer, uri)

	context.Destroy()
}

func makeDiagnostics(reports report.Reports, writer *bufio.Writer, uri string) {
	if reports == nil {
		log.Println("Compilation reports is nil")
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	// Group diagnostics by file URI
	diagnosticsByFile := make(map[string][]map[string]interface{})

	log.Printf("Found %d problems\n", len(reports))

	for _, report := range reports {
		// Convert file path to URI format
		fileURI := wio.PathToURI(report.FilePath)

		diagnostic := map[string]interface{}{
			"range": map[string]interface{}{
				"start": map[string]int{"line": report.Location.Start.Line - 1, "character": report.Location.Start.Column - 1},
				"end":   map[string]int{"line": report.Location.End.Line - 1, "character": report.Location.End.Column - 1},
			},
			"message":  report.Message,
			"severity": getSeverity(report.Level),
		}

		// Group diagnostics by file
		if diagnosticsByFile[fileURI] == nil {
			diagnosticsByFile[fileURI] = make([]map[string]interface{}, 0)
		}
		diagnosticsByFile[fileURI] = append(diagnosticsByFile[fileURI], diagnostic)
	}

	// Publish diagnostics for each file separately
	for fileURI, fileDiagnostics := range diagnosticsByFile {
		log.Printf("Publishing %d diagnostics for %s", len(fileDiagnostics), fileURI)
		publishDiagnostics(writer, fileURI, fileDiagnostics)
		filesWithDiagnostics[fileURI] = true
	}

	// Clear diagnostics for files that previously had problems but now don't
	for previousFileURI := range filesWithDiagnostics {
		if _, hasProblems := diagnosticsByFile[previousFileURI]; !hasProblems {
			log.Printf("Clearing diagnostics for %s (no longer has problems)", previousFileURI)
			publishDiagnostics(writer, previousFileURI, []map[string]interface{}{})
			delete(filesWithDiagnostics, previousFileURI)
		}
	}
}

func getSeverity(level report.PROBLEM_TYPE) int {
	switch level {
	case report.CRITICAL_ERROR, report.SYNTAX_ERROR, report.NORMAL_ERROR, report.SEMANTIC_ERROR:
		return 1
	case report.WARNING:
		return 2
	case report.INFO:
		return 3
	default:
		return 4
	}
}

func publishDiagnostics(writer *bufio.Writer, uri string, diagnostics []map[string]interface{}) {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params": map[string]interface{}{
			"uri":         uri,
			"diagnostics": diagnostics,
		},
	}
	writeRawMessage(writer, notification)
}
