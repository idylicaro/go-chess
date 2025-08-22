package debug

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// DebugLevel represents different levels of debugging
type DebugLevel int

const (
	DebugNone DebugLevel = iota
	DebugBasic
	DebugDetailed
	DebugVerbose
)

// DebugConfig holds the debug configuration
type DebugConfig struct {
	Enabled     bool
	Level       DebugLevel
	LogToFile   bool
	LogFilePath string
	ShowTiming  bool
	logger      *log.Logger
}

// Global debug instance
var Debug = &DebugConfig{
	Enabled:     false,
	Level:       DebugNone,
	LogToFile:   false,
	LogFilePath: "chess_debug.log",
	ShowTiming:  false,
}

// InitDebug initializes the debug system
func InitDebug(level DebugLevel, logToFile bool, logPath string) error {
	Debug.Enabled = level > DebugNone
	Debug.Level = level
	Debug.LogToFile = logToFile
	Debug.ShowTiming = level >= DebugDetailed

	if logToFile {
		Debug.LogFilePath = logPath
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open debug log file: %v", err)
		}
		Debug.logger = log.New(file, "", log.LstdFlags)
	} else {
		Debug.logger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
	}

	Debug.Log(DebugBasic, "Debug mode initialized - Level: %v, LogToFile: %v", level, logToFile)
	return nil
}

// Log logs a debug message if the level is appropriate
func (d *DebugConfig) Log(level DebugLevel, format string, args ...interface{}) {
	if !d.Enabled || level > d.Level {
		return
	}

	prefix := ""
	switch level {
	case DebugBasic:
		prefix = "[INFO] "
	case DebugDetailed:
		prefix = "[DETAIL] "
	case DebugVerbose:
		prefix = "[VERBOSE] "
	}

	message := fmt.Sprintf(prefix+format, args...)
	if d.logger != nil {
		d.logger.Println(message)
	}
}

// LogTiming logs timing information for a function
func (d *DebugConfig) LogTiming(functionName string, start time.Time) {
	if !d.Enabled || !d.ShowTiming {
		return
	}

	duration := time.Since(start)
	d.Log(DebugDetailed, "%s took %v", functionName, duration)
}

// LogError logs error information
func (d *DebugConfig) LogError(context string, err error) {
	if !d.Enabled {
		return
	}

	d.Log(DebugBasic, "ERROR in %s: %v", context, err)
}

// EnableFromEnvironment enables debug based on environment variables
func EnableFromEnvironment() error {
	debugLevel := os.Getenv("CHESS_DEBUG_LEVEL")
	debugFile := os.Getenv("CHESS_DEBUG_FILE")

	if debugLevel == "" {
		return nil // No debug requested
	}

	var level DebugLevel
	switch strings.ToLower(debugLevel) {
	case "basic", "1":
		level = DebugBasic
	case "detailed", "2":
		level = DebugDetailed
	case "verbose", "3":
		level = DebugVerbose
	default:
		return fmt.Errorf("invalid debug level: %s", debugLevel)
	}

	logToFile := debugFile != ""
	if !logToFile {
		debugFile = "chess_debug.log"
	}

	return InitDebug(level, logToFile, debugFile)
}

// Helper functions for easy logging
func LogInfo(format string, args ...interface{}) {
	Debug.Log(DebugBasic, format, args...)
}

func LogDetail(format string, args ...interface{}) {
	Debug.Log(DebugDetailed, format, args...)
}

func LogVerbose(format string, args ...interface{}) {
	Debug.Log(DebugVerbose, format, args...)
}

func LogError(context string, err error) {
	Debug.LogError(context, err)
}

func LogTiming(functionName string, start time.Time) {
	Debug.LogTiming(functionName, start)
}
