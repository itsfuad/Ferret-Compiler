package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/colors"
	"compiler/internal/source"

	"compiler/internal/utils"
)

type PROBLEM_TYPE string

type COMPILATION_PHASE string

const (
	LEXING_PHASE    COMPILATION_PHASE = "lexing"
	PARSING_PHASE   COMPILATION_PHASE = "parsing"
	COLLECTOR_PHASE COMPILATION_PHASE = "collecting symbols"
	RESOLVER_PHASE  COMPILATION_PHASE = "resolving"
	TYPECHECK_PHASE COMPILATION_PHASE = "type checking"
)

const (
	NULL           PROBLEM_TYPE = ""
	SEMANTIC_ERROR PROBLEM_TYPE = "semantic error" // Semantic error
	CRITICAL_ERROR PROBLEM_TYPE = "critical error" // Stops compilation immediately
	SYNTAX_ERROR   PROBLEM_TYPE = "syntax error"   // Syntax error, also stops compilation
	NORMAL_ERROR   PROBLEM_TYPE = "error"          // Regular error that doesn't halt compilation

	WARNING PROBLEM_TYPE = "warning" // Indicates potential issues
	INFO    PROBLEM_TYPE = "info"    // Informational message
)

// var colorMap = make(map[PROBLEM_TYPE]utils.COLOR)
var colorMap = map[PROBLEM_TYPE]colors.COLOR{
	CRITICAL_ERROR: colors.BRIGHT_RED,
	SYNTAX_ERROR:   colors.RED,
	SEMANTIC_ERROR: colors.RED,
	NORMAL_ERROR:   colors.RED,
	WARNING:        colors.YELLOW,
	INFO:           colors.BLUE,
}

type Reports []*Report

func (r Reports) Len() int {
	return len(r)
}
func (r *Reports) HasErrors() bool {
	for _, report := range *r {
		if report.Level == NORMAL_ERROR || report.Level == CRITICAL_ERROR || report.Level == SYNTAX_ERROR || report.Level == SEMANTIC_ERROR {
			return true
		}
	}
	return false
}
func (r *Reports) HasWarnings() bool {
	for _, report := range *r {
		if report.Level == WARNING {
			return true
		}
	}
	return false
}
func (r *Reports) ShouldStopCompilation() bool {
	for _, report := range *r {
		if report.ShouldStop {
			return true
		}
	}
	return false
}
func (r *Reports) DisplayAll() {

	fmt.Println()

	ln := len(*r) - 1
	for i, report := range *r {
		printReport(report)
		if i < ln {
			fmt.Print("\n\n")
		}
	}

	(*r).ShowStatus()
}

// Report represents a diagnostic report used both internally and by LSP.
type Report struct {
	FilePath   string
	Location   *source.Location
	Message    string
	Hint       string
	Label      string
	Level      PROBLEM_TYPE
	Phase      COMPILATION_PHASE
	ShouldStop bool // Flag to indicate if compilation should stop gracefully
}

// printReport prints a formatted diagnostic report to stdout.
// It shows file location, a code snippet, underline highlighting, any hints,
// and panics if the diagnostic level is critical or indicates a syntax error.
func printReport(r *Report) {
	fileData, err := os.ReadFile(filepath.FromSlash(r.FilePath))

	if os.IsNotExist(err) {
		panic(fmt.Sprintf("file %q not found", r.FilePath))
	}

	lines := strings.Split(string(fileData), "\n")
	currentLine := lines[r.Location.Start.Line-1]

	// Calculate the maximum line number width we'll need
	// This includes the current line and any previous lines we might show
	maxLineNum := r.Location.Start.Line
	lineNumWidth := len(fmt.Sprint(maxLineNum))

	// Calculate underline length
	hLen := 0
	if r.Location.Start.Line == r.Location.End.Line {
		hLen = (r.Location.End.Column - r.Location.Start.Column) - 1
	} else {
		//full line
		hLen = len(currentLine) - 2
	}
	if hLen < 0 {
		hLen = 0
	}

	// Create formatted line number and bar using consistent width
	lineNumberStr := fmt.Sprintf("%*d | ", lineNumWidth, r.Location.Start.Line)
	barStr := fmt.Sprintf("%s |", strings.Repeat(" ", lineNumWidth))

	// Calculate padding for underline
	padding := strings.Repeat(" ", r.Location.Start.Column)

	var reportMsgType string

	switch r.Level {
	case WARNING:
		reportMsgType = fmt.Sprintf("[Warning while %s 🚨]: ", r.Phase)
	case INFO:
		reportMsgType = fmt.Sprintf("[Info while %s 😓]: ", r.Phase)
	case CRITICAL_ERROR:
		reportMsgType = fmt.Sprintf("[Critical Error while %s 💀]: ", r.Phase)
	case SYNTAX_ERROR:
		reportMsgType = fmt.Sprintf("[Syntax Error while %s 😑]: ", r.Phase)
	case NORMAL_ERROR:
		reportMsgType = fmt.Sprintf("[Error while %s 😨]: ", r.Phase)
	case SEMANTIC_ERROR:
		reportMsgType = fmt.Sprintf("[Semantic Error while %s 😱]: ", r.Phase)
	}

	// Build snippet
	snippet := colors.GREY.Sprintln(barStr)
	addPrevLines(r, &snippet, lines, lineNumWidth)
	snippet += colors.WHITE.Sprint(lineNumberStr) + currentLine + "\n"
	snippet += colors.GREY.Sprint(barStr)

	reportColor := colorMap[r.Level]

	//numlen is the length of the line number
	numlen := len(fmt.Sprint(r.Location.Start.Line))

	// The error message type and the message itself are printed in the same color.
	reportColor.Print(reportMsgType)
	reportColor.Println(r.Message)
	colors.GREY.Printf("%s> [%s:%d:%d]\n", strings.Repeat("-", numlen+2), r.FilePath, r.Location.Start.Line, r.Location.Start.Column)

	// The code snippet and underline are printed in the same color.
	fmt.Print(snippet)
	underline := fmt.Sprintf("%s^%s", padding, strings.Repeat("~", hLen))

	if r.Label != "" {
		reportColor.Print(underline)
		colors.RED.Printf(" %s\n", r.Label)
	} else {
		reportColor.Println(underline)
	}

	if r.Hint != "" {
		colors.YELLOW.Printf("Help: %s\n", r.Hint)
	}

	//colors.GREY.Println(strings.Repeat("-", numlen+2))
}

func addPrevLines(r *Report, snippet *string, lines []string, lineNumWidth int) {
	col := colors.GREY

	// PrevLine1
	// PrevLine2
	// MainLine

	prevLines := []string{}
	// if pl1 is empty, do nothing
	if r.Location.Start.Line-2 >= 0 && strings.TrimSpace(lines[r.Location.Start.Line-2]) != "" {
		prevLines = append(prevLines, lines[r.Location.Start.Line-2])
	}
	// if has pl1, add pl2. else skip pl2 if empty
	if len(prevLines) == 1 && r.Location.Start.Line-3 >= 0 && strings.TrimSpace(lines[r.Location.Start.Line-3]) != "" {
		prevLines = append([]string{lines[r.Location.Start.Line-3]}, prevLines...)
	}

	for i, pl := range prevLines {
		*snippet += col.Sprint(fmt.Sprintf("%*d | ", lineNumWidth, r.Location.Start.Line-len(prevLines)+i)) + pl + "\n"
	}
}

// AddHint appends a new hint message to the diagnostic and returns the updated diagnostic.
// It ignores empty hint messages.
func (r *Report) AddLabel(msg string) *Report {

	if msg == "" {
		return r
	}

	r.Label = msg

	return r
}

func (r *Report) AddHint(msg string) *Report {

	if msg == "" {
		return r
	}

	r.Hint = msg

	return r
}

// createNew creates and registers a new diagnostic report with basic position validation.
// It returns a pointer to the newly created Diagnostic.
func (r *Reports) createNew(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {

	if location.Start.Line < 1 {
		location.Start.Line = 1
	}
	if location.End.Line < 1 {
		location.End.Line = 1
	}
	if location.Start.Column < 1 {
		location.Start.Column = 1
	}
	if location.End.Column < 1 {
		location.End.Column = 1
	}

	report := &Report{
		FilePath: filePath,
		Location: location,
		Message:  msg,
		Level:    NULL,
		Phase:    phase,
	}

	if len(*r) == 0 {
		*r = make([]*Report, 0, 10) // Initialize with a capacity of 10
	}

	*r = append(*r, report)

	return report
}

func (r *Report) setLevel(level PROBLEM_TYPE) {
	if level == NULL {
		panic("invalid Error level")
	}
	r.Level = level
	if level == CRITICAL_ERROR || level == SYNTAX_ERROR {
		r.ShouldStop = true // Set flag instead of panicking
	}
}

// AddError creates and registers a new error report
func (r *Reports) AddError(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {
	report := r.createNew(filePath, location, msg, phase)
	report.setLevel(NORMAL_ERROR)
	return report
}

// AddSemanticError creates and registers a new semantic error report
func (r *Reports) AddSemanticError(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {
	report := r.createNew(filePath, location, msg, phase)
	report.setLevel(SEMANTIC_ERROR)
	return report
}

// AddSyntaxError creates and registers a new syntax error report
func (r *Reports) AddSyntaxError(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {
	report := r.createNew(filePath, location, msg, phase)
	report.setLevel(SYNTAX_ERROR)
	return report
}

// AddCriticalError creates and registers a new critical error report
func (r *Reports) AddCriticalError(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {
	report := r.createNew(filePath, location, msg, phase)
	report.setLevel(CRITICAL_ERROR)
	return report
}

// AddWarning creates and registers a new warning report
func (r *Reports) AddWarning(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {
	report := r.createNew(filePath, location, msg, phase)
	report.setLevel(WARNING)
	return report
}

// AddInfo creates and registers a new info report
func (r *Reports) AddInfo(filePath string, location *source.Location, msg string, phase COMPILATION_PHASE) *Report {
	report := r.createNew(filePath, location, msg, phase)
	report.setLevel(INFO)
	return report
}

// ShowStatus displays a summary of compilation status along with counts of warnings and errors.
func (r Reports) ShowStatus() {
	warningCount := 0
	probCount := 0

	for _, report := range r {
		switch report.Level {
		case WARNING:
			warningCount++
		case NORMAL_ERROR, CRITICAL_ERROR, SYNTAX_ERROR, SEMANTIC_ERROR:
			probCount++
		}
	}

	var messageColor colors.COLOR

	if probCount > 0 {
		messageColor = colors.RED
		messageColor.Print("------------- failed ")
	} else {
		messageColor = colors.GREEN
		messageColor.Print("------------- Passed ")
	}

	totalProblemsString := ""

	// Example combinations:
	// -- Passed -- // No error or warning
	// -- Passed with N warnings -- // No error, just N warnings
	// -- Failed with N errors -- // No warning, just N errors
	// -- Passed with N warnings and M errors -- // N warnings, M errors

	if warningCount > 0 && probCount == 0 {
		totalProblemsString = fmt.Sprintf("with %d %s ", warningCount, utils.Ternary(warningCount == 1, "warning", "warnings"))
	} else if probCount > 0 && warningCount == 0 {
		totalProblemsString = fmt.Sprintf("with %d %s ", probCount, utils.Ternary(probCount == 1, "error", "errors"))
	} else if probCount > 0 && warningCount > 0 {
		totalProblemsString = fmt.Sprintf("with %d %s and %d %s ", warningCount, utils.Ternary(warningCount == 1, "warning", "warnings"), probCount, utils.Ternary(probCount == 1, "error", "errors"))
	}

	messageColor.Print(totalProblemsString)
	messageColor.Println("-------------")

	// Check for critical errors and exit gracefully
	for _, report := range r {
		if report.ShouldStop {
			os.Exit(1) // Graceful exit instead of panic
		}
	}

	//os.Exit(errCode)
}
