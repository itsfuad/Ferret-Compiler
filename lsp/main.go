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

// Track files that have had diagnostics published and their analysis mode
var filesWithDiagnostics = make(map[string]bool)
var fileAnalysisMode = make(map[string]string) // "project" or "single"

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
	var connectionClosed bool

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleConnection: %v", r)
		}
		if !connectionClosed {
			conn.Close()
			connectionClosed = true
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

	// flush pending messages safely
	if writer != nil {
		if err := writer.Flush(); err != nil {
			log.Printf("Warning: Could not flush buffer on exit: %v", err)
		}
	}

	// Don't close connection here - let the defer handle it
	log.Println("Exit request processed")
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in writeMessage: %v", r)
		}
	}()

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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in writeRawMessage: %v", r)
		}
	}()

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

// hasFileInReports checks if a specific file has any reports/diagnostics
func hasFileInReports(reports report.Reports, filePath string) bool {
	if reports == nil {
		return false
	}

	for _, r := range reports {
		if r.FilePath == filePath {
			return true
		}
	}
	return false
}

func processDiagnostics(writer *bufio.Writer, uri string) {
	defer recoverFromProcessingPanic(writer, uri)

	log.Println("Processing diagnostics for:", uri)

	filePath, err := validateFile(uri, writer)
	if err != nil {
		return
	}

	log.Println("File path:", filePath)

	// Try project-based analysis first
	if tryProjectBasedAnalysis(writer, uri, filePath) {
		return
	}

	// Fall back to single-file analysis
	trySingleFileAnalysis(writer, uri, filePath)
}

func recoverFromProcessingPanic(writer *bufio.Writer, uri string) {
	if r := recover(); r != nil {
		log.Printf("Panic in processDiagnostics: %v", r)
		safelySendEmptyDiagnostics(writer, uri)
	}
}

func safelySendEmptyDiagnostics(writer *bufio.Writer, uri string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Failed to send error diagnostics: %v", r)
		}
	}()
	publishDiagnostics(writer, uri, []map[string]interface{}{})
}

func validateFile(uri string, writer *bufio.Writer) (string, error) {
	filePath, err := wio.UriToFilePath(uri)
	if err != nil {
		log.Println("Error converting URI to file path:", err)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return "", err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", filePath)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return "", err
	}

	return filePath, nil
}

func tryProjectBasedAnalysis(writer *bufio.Writer, uri, filePath string) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in tryProjectBasedAnalysis: %v", r)
		}
	}()

	if uri == "" || filePath == "" {
		log.Printf("Empty URI or file path provided")
		return false
	}

	projectRoot, err := config.GetProjectRoot(filePath)
	if err != nil {
		log.Printf("No project root found for %s: %v", filePath, err)
		return false
	}

	log.Printf("Found project root for %s: %s", filePath, projectRoot)
	result := cmd.CompileProjectForLSP(projectRoot, false)

	if result == nil {
		log.Printf("Project analysis returned nil result for %s", projectRoot)
		return false
	}

	log.Printf("Project analysis completed - Success: %v, Reports: %d", result.Success, len(result.Reports))

	if !result.Success && len(result.Reports) == 0 {
		log.Printf("Project analysis failed and produced no reports for %s", projectRoot)
		return false
	}

	if !hasFileInReports(result.Reports, filePath) {
		log.Printf("File %s not found in project analysis results (checked %d reports)", filePath, len(result.Reports))
		// Log the files that were found in reports for debugging
		filePathsInReports := make([]string, 0)
		for _, report := range result.Reports {
			filePathsInReports = append(filePathsInReports, report.FilePath)
		}
		log.Printf("Files found in reports: %v", filePathsInReports)
		return false
	}

	log.Printf("Project-based analysis successful for %s, processing %d reports", filePath, len(result.Reports))

	// Mark all files in this project analysis as "project" mode - safely
	for _, report := range result.Reports {
		if report.FilePath != "" {
			reportURI := wio.PathToURI(report.FilePath)
			if reportURI != "" {
				fileAnalysisMode[reportURI] = "project"
			}
		}
	}

	makeDiagnostics(result.Reports, writer, uri)
	return true
}

func trySingleFileAnalysis(writer *bufio.Writer, uri, filePath string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in trySingleFileAnalysis: %v", r)
		}
	}()

	if uri == "" || filePath == "" {
		log.Printf("Empty URI or file path provided to single file analysis")
		return
	}

	log.Println("Falling back to single-file analysis")
	result := cmd.CompileForLSP(filePath, false)

	// Mark this file as "single" mode - safely
	if uri != "" {
		fileAnalysisMode[uri] = "single"
	}

	if result != nil {
		makeDiagnostics(result.Reports, writer, uri)
	} else {
		log.Printf("Single-file analysis failed for %s", filePath)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
	}
}

func makeDiagnostics(reports report.Reports, writer *bufio.Writer, uri string) {
	if reports == nil {
		log.Println("Compilation reports is nil")
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		return
	}

	// Group diagnostics by file URI
	diagnosticsByFile := createDiagnosticsByFile(reports)
	log.Printf("Found %d problems grouped into %d files\n", len(reports), len(diagnosticsByFile))

	// Publish diagnostics for each file separately
	publishDiagnosticsForFiles(writer, diagnosticsByFile)

	// Clear diagnostics for files that no longer have problems
	clearOldDiagnostics(writer, diagnosticsByFile, uri)
}

func createDiagnosticsByFile(reports report.Reports) map[string][]map[string]interface{} {
	diagnosticsByFile := make(map[string][]map[string]interface{})

	for _, report := range reports {
		fileURI := wio.PathToURI(report.FilePath)
		diagnostic := map[string]interface{}{
			"range": map[string]interface{}{
				"start": map[string]int{"line": report.Location.Start.Line - 1, "character": report.Location.Start.Column - 1},
				"end":   map[string]int{"line": report.Location.End.Line - 1, "character": report.Location.End.Column - 1},
			},
			"message":  report.Message,
			"severity": getSeverity(report.Level),
		}

		if diagnosticsByFile[fileURI] == nil {
			diagnosticsByFile[fileURI] = make([]map[string]interface{}, 0)
		}
		diagnosticsByFile[fileURI] = append(diagnosticsByFile[fileURI], diagnostic)
	}

	return diagnosticsByFile
}

func publishDiagnosticsForFiles(writer *bufio.Writer, diagnosticsByFile map[string][]map[string]interface{}) {
	for fileURI, fileDiagnostics := range diagnosticsByFile {
		log.Printf("Publishing %d diagnostics for %s", len(fileDiagnostics), fileURI)
		publishDiagnostics(writer, fileURI, fileDiagnostics)
		filesWithDiagnostics[fileURI] = true
	}
}

func clearOldDiagnostics(writer *bufio.Writer, diagnosticsByFile map[string][]map[string]interface{}, uri string) {
	if len(diagnosticsByFile) == 1 {
		clearSingleFileDiagnostics(writer, diagnosticsByFile, uri)
	} else {
		clearProjectFileDiagnostics(writer, diagnosticsByFile)
	}
}

func clearSingleFileDiagnostics(writer *bufio.Writer, diagnosticsByFile map[string][]map[string]interface{}, uri string) {
	if len(diagnosticsByFile[uri]) == 0 {
		log.Printf("Clearing diagnostics for single file %s (no problems)", uri)
		publishDiagnostics(writer, uri, []map[string]interface{}{})
		delete(filesWithDiagnostics, uri)
	}
}

func clearProjectFileDiagnostics(writer *bufio.Writer, diagnosticsByFile map[string][]map[string]interface{}) {
	for previousFileURI := range filesWithDiagnostics {
		if shouldClearFile(previousFileURI, diagnosticsByFile) {
			log.Printf("Clearing diagnostics for project file %s (no longer has problems)", previousFileURI)
			publishDiagnostics(writer, previousFileURI, []map[string]interface{}{})
			delete(filesWithDiagnostics, previousFileURI)
			delete(fileAnalysisMode, previousFileURI)
		}
	}
}

func shouldClearFile(fileURI string, diagnosticsByFile map[string][]map[string]interface{}) bool {
	if fileURI == "" {
		return false
	}
	mode, exists := fileAnalysisMode[fileURI]
	_, hasProblems := diagnosticsByFile[fileURI]
	return exists && mode == "project" && !hasProblems
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
