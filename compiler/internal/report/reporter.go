package report

import (
	"compiler/colors"
	"compiler/internal/source"

	//"compiler/internal/symboltable"
	"compiler/internal/utils"
	"fmt"
	"os"
	"strings"
)

type REPORT_TYPE string

const (
	NULL           REPORT_TYPE = ""
	SEMANTIC_ERROR REPORT_TYPE = "semantic error" // Semantic error
	CRITICAL_ERROR REPORT_TYPE = "critical error" // Stops compilation immediately
	SYNTAX_ERROR   REPORT_TYPE = "syntax error"   // Syntax error, also stops compilation
	NORMAL_ERROR   REPORT_TYPE = "error"          // Regular error that doesn't halt compilation

	WARNING REPORT_TYPE = "warning" // Indicates potential issues
	INFO    REPORT_TYPE = "info"    // Informational message
)

// var colorMap = make(map[REPORT_TYPE]utils.COLOR)
var colorMap = map[REPORT_TYPE]colors.COLOR{
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
	if len(*r) == 0 {
		return
	}

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
	Level    REPORT_TYPE
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
		reportMsgType = "[Warning 🚨]: "
	case INFO:
		reportMsgType = "[Info 😓]: "
	case CRITICAL_ERROR:
		reportMsgType = "[Critical Error 💀]: "
	case SYNTAX_ERROR:
		reportMsgType = "[Syntax Error 😑]: "
	case NORMAL_ERROR:
		reportMsgType = "[Error 😨]: "
	case SEMANTIC_ERROR:
		reportMsgType = "[Semantic Error 😱]: "
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
	fileData, err := os.ReadFile(r.FilePath)
	if err != nil {
		panic(err)
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

// Add creates and registers a new diagnostic report with basic position validation.
// It returns a pointer to the newly created Diagnostic.
func (r *Reports) Add(filePath string, location *source.Location, msg string) *Report {

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
	}

	if len(*r) == 0 {
		*r = make([]*Report, 0, 10) // Initialize with a capacity of 10
	}

	*r = append(*r, report)

	return report
}

// SetLevel assigns a diagnostic level to the report, increments its count,
// and triggers DisplayAll if the level is critical or denotes a syntax error.
func (e *Report) SetLevel(level REPORT_TYPE) {
	if level == NULL {
		panic("call SetLevel() method with valid Error level")
	}
	e.Level = level
	if level == CRITICAL_ERROR || level == SYNTAX_ERROR {
		panic("critical or syntax error encountered, stopping compilation")
	}
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
		totalProblemsString += colorMap[WARNING].Sprintf("(%d %s)", warningCount, utils.Plural("warning", "warnings ", warningCount))
		if probCount > 0 {
			totalProblemsString += colors.ORANGE.Sprintf(", ")
		}
	}

	if probCount > 0 {
		totalProblemsString += colorMap[NORMAL_ERROR].Sprintf("%d %s", probCount, utils.Plural("error", "errors", probCount))
	}

	messageColor.Print(totalProblemsString)
	messageColor.Println(" -------------")
}
