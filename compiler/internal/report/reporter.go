package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ferret/compiler/colors"
	"ferret/compiler/internal/source"

	//"ferret/compiler/internal/symboltable"
	_strings "ferret/compiler/internal/utils/strings"
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
	CRITICAL_ERROR: colors.BOLD_RED,
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
func (r *Reports) DisplayAll() {

	for _, report := range *r {
		printReport(report)
	}

	(*r).ShowStatus()
}

type HintContainer struct {
	hint string
	col  int
}

// Report represents a diagnostic report used both internally and by LSP.
type Report struct {
	FilePath string
	Location *source.Location
	Message  string
	Hints    HintContainer
	Level    PROBLEM_TYPE
	Phase    COMPILATION_PHASE
}

// printReport prints a formatted diagnostic report to stdout.
// It shows file location, a code snippet, underline highlighting, any hints,
// and panics if the diagnostic level is critical or indicates a syntax error.
func printReport(r *Report) {

	// Generate the code snippet and underline.
	// hLen is the padding length for hint messages.
	snippet, underline := makeParts(r)

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

	reportColor := colorMap[r.Level]

	// The error message type and the message itself are printed in the same color.
	reportColor.Print(reportMsgType)
	reportColor.Println(r.Message)

	//numlen is the length of the line number
	numlen := len(fmt.Sprint(r.Location.Start.Line))

	// The file path is printed in grey.
	colors.GREY.Printf("%s> [%s:%d:%d]\n", strings.Repeat("-", numlen+2), r.FilePath, r.Location.Start.Line, r.Location.Start.Column)

	// The code snippet and underline are printed in the same color.
	fmt.Print(snippet)

	if r.Hints.hint != "" {
		reportColor.Print(underline)
		colors.YELLOW.Printf(" %s%s\n", r.Hints.hint, strings.Repeat(" ", r.Location.Start.Column-r.Hints.col))
	} else {
		reportColor.Println(underline)
	}
}

// makeParts reads the source file and generates a code snippet and underline
// indicating the location of the diagnostic. It returns the snippet, underline,
// and a padding value.
func makeParts(r *Report) (snippet, underline string) {
	fileData, err := os.ReadFile(filepath.FromSlash(r.FilePath))

	if os.IsNotExist(err) {
		panic(fmt.Sprintf("file '%s' not found", r.FilePath))
	}

	lines := strings.Split(string(fileData), "\n")
	line := lines[r.Location.Start.Line-1]

	hLen := 0

	if r.Location.Start.Line == r.Location.End.Line {
		hLen = (r.Location.End.Column - r.Location.Start.Column) - 1
	} else {
		//full line
		hLen = len(line) - 2
	}
	if hLen < 0 {
		hLen = 0
	}

	bar := fmt.Sprintf("%s |", strings.Repeat(" ", len(fmt.Sprint(r.Location.Start.Line))))
	lineNumber := fmt.Sprintf("%d | ", r.Location.Start.Line)

	padding := strings.Repeat(" ", (((r.Location.Start.Column - 1) + len(lineNumber)) - len(bar)))

	snippet = colors.GREY.Sprint(bar) + "\n" + colors.GREY.Sprint(lineNumber) + line + "\n"
	snippet += colors.GREY.Sprint(bar)
	underline = fmt.Sprintf("%s^%s", padding, strings.Repeat("~", hLen))

	return snippet, underline
}

// AddHint appends a new hint message to the diagnostic and returns the updated diagnostic.
// It ignores empty hint messages.
func (r *Report) AddHint(msg string) *Report {

	if msg == "" {
		return r
	}

	r.Hints.hint = msg
	r.Hints.col = r.Location.Start.Column

	return r
}

func (r *Report) AddHintAt(msg string, col int) *Report {
	if msg == "" {
		return r
	}

	r.Hints.hint = msg

	if col < r.Location.Start.Column {
		col = r.Location.Start.Column
	}

	r.Hints.col = col

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

func (e *Report) setLevel(level PROBLEM_TYPE) {
	if level == NULL {
		panic("call SetLevel() method with valid Error level")
	}
	e.Level = level
	if level == CRITICAL_ERROR || level == SYNTAX_ERROR {
		panic("critical or syntax error encountered, stopping compilation")
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
		messageColor.Print("------------- failed with ")
	} else {
		messageColor = colors.GREEN
		messageColor.Print("------------- Passed ")
	}

	totalProblemsString := ""

	if warningCount > 0 {
		totalProblemsString += colorMap[WARNING].Sprintf("(%d %s) ", warningCount, _strings.Plural("warning", "warnings ", warningCount))
		if probCount > 0 {
			totalProblemsString += colors.ORANGE.Sprintf(", ")
		}
	}

	//errCode := 0

	if probCount > 0 {
		//errCode = -1
		totalProblemsString += colorMap[NORMAL_ERROR].Sprintf("%d %s", probCount, _strings.Plural("error", "errors", probCount))
	}

	messageColor.Print(totalProblemsString)
	messageColor.Println("-------------")

	//os.Exit(errCode)
}
