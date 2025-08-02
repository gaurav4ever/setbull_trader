package v2

import (
	"context"
	"fmt"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DebugConsole provides interactive debugging commands
type DebugConsole struct {
	debugManager *DebugManager
	logger       DebugLogger
	running      bool
	mu           sync.RWMutex
}

// DebugCommand represents a debug command
type DebugCommand struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Usage       string                     `json:"usage"`
	Handler     func(args []string) string `json:"-"`
}

// NewDebugConsole creates a new debug console
func NewDebugConsole(debugManager *DebugManager, logger DebugLogger) *DebugConsole {
	dc := &DebugConsole{
		debugManager: debugManager,
		logger:       logger,
		running:      false,
	}

	// Log console creation
	if logger != nil {
		helper := NewLogHelper(logger, GenerateTraceID(), "debug_console")
		helper.LogInfo(SYSTEM, "DebugConsole", "NewDebugConsole", "", "",
			"Debug console created", nil)
	}

	return dc
}

// Start starts the debug console
func (dc *DebugConsole) Start(ctx context.Context) {
	dc.mu.Lock()
	if dc.running {
		dc.mu.Unlock()
		return
	}
	dc.running = true
	dc.mu.Unlock()

	if dc.logger != nil {
		helper := NewLogHelper(dc.logger, GenerateTraceID(), "debug_console")
		helper.LogInfo(SYSTEM, "DebugConsole", "Start", "", "",
			"Debug console started", nil)
	}

	fmt.Println("🔧 V2 Strategy Engine Debug Console")
	fmt.Println("Type 'help' for available commands")
	fmt.Println("Type 'exit' to stop the console")
	fmt.Println()

	// Start command processing loop
	go dc.processCommands(ctx)
}

// Stop stops the debug console
func (dc *DebugConsole) Stop() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if !dc.running {
		return
	}

	dc.running = false

	if dc.logger != nil {
		helper := NewLogHelper(dc.logger, GenerateTraceID(), "debug_console")
		helper.LogInfo(SYSTEM, "DebugConsole", "Stop", "", "",
			"Debug console stopped", nil)
	}

	fmt.Println("Debug console stopped")
}

// IsRunning checks if the console is running
func (dc *DebugConsole) IsRunning() bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.running
}

// processCommands processes debug commands
func (dc *DebugConsole) processCommands(ctx context.Context) {
	commands := dc.getCommands()

	for {
		select {
		case <-ctx.Done():
			dc.Stop()
			return
		default:
			if !dc.IsRunning() {
				return
			}

			fmt.Print("🔧 debug> ")
			var input string
			fmt.Scanln(&input)

			if input == "exit" {
				dc.Stop()
				return
			}

			parts := strings.Fields(input)
			if len(parts) == 0 {
				continue
			}

			command := parts[0]
			args := parts[1:]

			// Find and execute command
			output := dc.executeCommand(commands, command, args)
			if output != "" {
				fmt.Println(output)
			}
		}
	}
}

// getCommands returns available debug commands
func (dc *DebugConsole) getCommands() map[string]*DebugCommand {
	return map[string]*DebugCommand{
		"help": {
			Name:        "help",
			Description: "Show available commands",
			Usage:       "help [command]",
			Handler:     dc.cmdHelp,
		},
		"status": {
			Name:        "status",
			Description: "Show debug system status",
			Usage:       "status",
			Handler:     dc.cmdStatus,
		},
		"enable": {
			Name:        "enable",
			Description: "Enable debug features",
			Usage:       "enable [feature]",
			Handler:     dc.cmdEnable,
		},
		"disable": {
			Name:        "disable",
			Description: "Disable debug features",
			Usage:       "disable [feature]",
			Handler:     dc.cmdDisable,
		},
		"snapshot": {
			Name:        "snapshot",
			Description: "Take system state snapshot",
			Usage:       "snapshot [component]",
			Handler:     dc.cmdSnapshot,
		},
		"memory": {
			Name:        "memory",
			Description: "Show memory usage",
			Usage:       "memory [detailed]",
			Handler:     dc.cmdMemory,
		},
		"performance": {
			Name:        "performance",
			Description: "Show performance metrics",
			Usage:       "performance [component]",
			Handler:     dc.cmdPerformance,
		},
		"errors": {
			Name:        "errors",
			Description: "Show error statistics",
			Usage:       "errors [component]",
			Handler:     dc.cmdErrors,
		},
		"progress": {
			Name:        "progress",
			Description: "Show progress tracking",
			Usage:       "progress [id]",
			Handler:     dc.cmdProgress,
		},
		"goroutines": {
			Name:        "goroutines",
			Description: "Show goroutine information",
			Usage:       "goroutines",
			Handler:     dc.cmdGoroutines,
		},
		"profile": {
			Name:        "profile",
			Description: "Generate performance profile",
			Usage:       "profile [cpu|memory|goroutine] [duration]",
			Handler:     dc.cmdProfile,
		},
		"gc": {
			Name:        "gc",
			Description: "Force garbage collection",
			Usage:       "gc",
			Handler:     dc.cmdGC,
		},
		"summary": {
			Name:        "summary",
			Description: "Show comprehensive debug summary",
			Usage:       "summary",
			Handler:     dc.cmdSummary,
		},
	}
}

// executeCommand executes a debug command
func (dc *DebugConsole) executeCommand(commands map[string]*DebugCommand, command string, args []string) string {
	cmd, exists := commands[command]
	if !exists {
		return fmt.Sprintf("❌ Unknown command: %s. Type 'help' for available commands.", command)
	}

	return cmd.Handler(args)
}

// cmdHelp handles the help command
func (dc *DebugConsole) cmdHelp(args []string) string {
	commands := dc.getCommands()

	if len(args) == 0 {
		// Show all commands
		output := "📚 Available Commands:\n\n"
		for _, cmd := range commands {
			output += fmt.Sprintf("  %-15s %s\n", cmd.Name, cmd.Description)
		}
		output += "\nType 'help <command>' for detailed usage."
		return output
	}

	// Show specific command help
	command := args[0]
	if cmd, exists := commands[command]; exists {
		return fmt.Sprintf("📖 %s\n\nDescription: %s\nUsage: %s", cmd.Name, cmd.Description, cmd.Usage)
	}

	return fmt.Sprintf("❌ Unknown command: %s", command)
}

// cmdStatus handles the status command
func (dc *DebugConsole) cmdStatus(args []string) string {
	flags := dc.debugManager.GetDebugFlags()

	output := "📊 Debug System Status:\n\n"
	output += fmt.Sprintf("🔧 Debug Enabled: %t\n", flags.Enabled)
	output += fmt.Sprintf("🔍 Trace Active: %t\n", flags.TraceActive)
	output += fmt.Sprintf("📸 State Inspection: %t\n", flags.StateInspection)
	output += fmt.Sprintf("💾 Memory Inspection: %t\n", flags.MemoryInspection)
	output += fmt.Sprintf("⚡ Performance Profiling: %t\n", flags.PerformanceProfiling)
	output += fmt.Sprintf("❌ Error Tracking: %t\n", flags.ErrorTracking)
	output += fmt.Sprintf("📈 Progress Tracking: %t\n", flags.ProgressTracking)

	// Add runtime info
	output += fmt.Sprintf("\n🖥️  Runtime Info:\n")
	output += fmt.Sprintf("  Goroutines: %d\n", runtime.NumGoroutine())

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	output += fmt.Sprintf("  Memory Alloc: %.2f MB\n", float64(memStats.Alloc)/1024/1024)
	output += fmt.Sprintf("  Memory Sys: %.2f MB\n", float64(memStats.Sys)/1024/1024)

	return output
}

// cmdEnable handles the enable command
func (dc *DebugConsole) cmdEnable(args []string) string {
	if len(args) == 0 {
		dc.debugManager.EnableDebug()
		return "✅ All debug features enabled"
	}

	feature := args[0]
	dc.debugManager.SetDebugFlag(feature, true)
	return fmt.Sprintf("✅ Debug feature '%s' enabled", feature)
}

// cmdDisable handles the disable command
func (dc *DebugConsole) cmdDisable(args []string) string {
	if len(args) == 0 {
		dc.debugManager.DisableDebug()
		return "❌ All debug features disabled"
	}

	feature := args[0]
	dc.debugManager.SetDebugFlag(feature, false)
	return fmt.Sprintf("❌ Debug feature '%s' disabled", feature)
}

// cmdSnapshot handles the snapshot command
func (dc *DebugConsole) cmdSnapshot(args []string) string {
	if len(args) == 0 {
		// Take general system snapshot
		dc.debugManager.TakeStateSnapshot("system", map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"command":   "snapshot",
		})
		return "📸 System state snapshot taken"
	}

	component := args[0]
	dc.debugManager.TakeStateSnapshot(component, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"command":   "snapshot",
		"component": component,
	})
	return fmt.Sprintf("📸 State snapshot taken for component: %s", component)
}

// cmdMemory handles the memory command
func (dc *DebugConsole) cmdMemory(args []string) string {
	dc.debugManager.TakeMemorySnapshot()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	output := "💾 Memory Usage:\n\n"
	output += fmt.Sprintf("Alloc:      %.2f MB\n", float64(memStats.Alloc)/1024/1024)
	output += fmt.Sprintf("TotalAlloc: %.2f MB\n", float64(memStats.TotalAlloc)/1024/1024)
	output += fmt.Sprintf("Sys:        %.2f MB\n", float64(memStats.Sys)/1024/1024)
	output += fmt.Sprintf("HeapAlloc:  %.2f MB\n", float64(memStats.HeapAlloc)/1024/1024)
	output += fmt.Sprintf("HeapSys:    %.2f MB\n", float64(memStats.HeapSys)/1024/1024)
	output += fmt.Sprintf("HeapObjects: %d\n", memStats.HeapObjects)
	output += fmt.Sprintf("NumGC:      %d\n", memStats.NumGC)

	if len(args) > 0 && args[0] == "detailed" {
		output += fmt.Sprintf("\n📊 Detailed Memory Info:\n")
		output += fmt.Sprintf("HeapIdle:     %.2f MB\n", float64(memStats.HeapIdle)/1024/1024)
		output += fmt.Sprintf("HeapInuse:    %.2f MB\n", float64(memStats.HeapInuse)/1024/1024)
		output += fmt.Sprintf("HeapReleased: %.2f MB\n", float64(memStats.HeapReleased)/1024/1024)
		output += fmt.Sprintf("StackInuse:   %.2f MB\n", float64(memStats.StackInuse)/1024/1024)
		output += fmt.Sprintf("StackSys:     %.2f MB\n", float64(memStats.StackSys)/1024/1024)
	}

	return output
}

// cmdPerformance handles the performance command
func (dc *DebugConsole) cmdPerformance(args []string) string {
	stats := dc.debugManager.GetPerformanceStats()

	if len(stats) == 0 {
		return "📊 No performance metrics available"
	}

	output := "⚡ Performance Metrics:\n\n"

	if len(args) > 0 {
		// Show specific component
		component := args[0]
		output += fmt.Sprintf("Component: %s\n\n", component)

		for key, metric := range stats {
			if strings.HasPrefix(key, component+":") {
				output += fmt.Sprintf("  %s:\n", metric.Method)
				output += fmt.Sprintf("    Calls: %d\n", metric.TotalCalls)
				output += fmt.Sprintf("    Avg:   %v\n", metric.AvgDuration)
				output += fmt.Sprintf("    Min:   %v\n", metric.MinDuration)
				output += fmt.Sprintf("    Max:   %v\n", metric.MaxDuration)
				output += fmt.Sprintf("    Last:  %v\n", metric.LastCall.Format("15:04:05"))
				output += "\n"
			}
		}
	} else {
		// Show all components
		for key, metric := range stats {
			output += fmt.Sprintf("%s:\n", key)
			output += fmt.Sprintf("  Calls: %d\n", metric.TotalCalls)
			output += fmt.Sprintf("  Avg:   %v\n", metric.AvgDuration)
			output += fmt.Sprintf("  Min:   %v\n", metric.MinDuration)
			output += fmt.Sprintf("  Max:   %v\n", metric.MaxDuration)
			output += "\n"
		}
	}

	return output
}

// cmdErrors handles the errors command
func (dc *DebugConsole) cmdErrors(args []string) string {
	errorStats := dc.debugManager.GetErrorStats()

	if len(errorStats) == 0 {
		return "❌ No error statistics available"
	}

	output := "❌ Error Statistics:\n\n"

	totalErrors := errorStats["total_errors"].(int)
	output += fmt.Sprintf("Total Errors: %d\n\n", totalErrors)

	if errorCounts, exists := errorStats["error_counts"]; exists {
		counts := errorCounts.(map[string]int)
		output += "Error Counts:\n"
		for errorKey, count := range counts {
			output += fmt.Sprintf("  %s: %d\n", errorKey, count)
		}
	}

	return output
}

// cmdProgress handles the progress command
func (dc *DebugConsole) cmdProgress(args []string) string {
	if len(args) > 0 {
		// Show specific progress
		id := args[0]
		progress := dc.debugManager.GetProgressInfo(id)

		if progress == nil {
			return fmt.Sprintf("❌ No progress found for ID: %s", id)
		}

		output := fmt.Sprintf("📈 Progress: %s\n\n", id)
		output += fmt.Sprintf("Component:      %s\n", progress.Component)
		output += fmt.Sprintf("Status:         %s\n", progress.Status)
		output += fmt.Sprintf("Progress:       %.2f%%\n", progress.Progress)
		output += fmt.Sprintf("Completed:      %d/%d\n", progress.CompletedItems, progress.TotalItems)
		output += fmt.Sprintf("Start Time:     %s\n", progress.StartTime.Format("15:04:05"))
		output += fmt.Sprintf("Last Update:    %s\n", progress.LastUpdate.Format("15:04:05"))

		if !progress.ETA.IsZero() {
			output += fmt.Sprintf("ETA:            %s\n", progress.ETA.Format("15:04:05"))
		}

		return output
	}

	// Show all progress
	allProgress := dc.debugManager.GetAllProgressInfo()

	if len(allProgress) == 0 {
		return "📈 No progress tracking available"
	}

	output := "📈 Progress Tracking:\n\n"
	for id, progress := range allProgress {
		output += fmt.Sprintf("%s:\n", id)
		output += fmt.Sprintf("  Component: %s\n", progress.Component)
		output += fmt.Sprintf("  Status:    %s\n", progress.Status)
		output += fmt.Sprintf("  Progress:  %.2f%%\n", progress.Progress)
		output += fmt.Sprintf("  Completed: %d/%d\n", progress.CompletedItems, progress.TotalItems)
		output += "\n"
	}

	return output
}

// cmdGoroutines handles the goroutines command
func (dc *DebugConsole) cmdGoroutines(args []string) string {
	output := "🔄 Goroutine Information:\n\n"
	output += fmt.Sprintf("Active Goroutines: %d\n", runtime.NumGoroutine())

	// Get goroutine profile
	profile := pprof.Lookup("goroutine")
	if profile != nil {
		output += "\nGoroutine Profile:\n"
		// Note: In a real implementation, you might want to write this to a file
		// or format it differently for console output
		output += "  (Use 'profile goroutine' for detailed profile)\n"
	}

	return output
}

// cmdProfile handles the profile command
func (dc *DebugConsole) cmdProfile(args []string) string {
	if len(args) == 0 {
		return "❌ Profile type required. Usage: profile [cpu|memory|goroutine] [duration]"
	}

	profileType := args[0]
	duration := 30 // default 30 seconds

	if len(args) > 1 {
		if d, err := strconv.Atoi(args[1]); err == nil {
			duration = d
		}
	}

	output := fmt.Sprintf("📊 Generating %s profile for %d seconds...\n", profileType, duration)

	// In a real implementation, you would:
	// 1. Start the profile
	// 2. Wait for the duration
	// 3. Stop the profile
	// 4. Write to a file

	switch profileType {
	case "cpu":
		output += "CPU profile would be written to cpu.prof"
	case "memory":
		output += "Memory profile would be written to memory.prof"
	case "goroutine":
		output += "Goroutine profile would be written to goroutine.prof"
	default:
		return fmt.Sprintf("❌ Unknown profile type: %s", profileType)
	}

	return output
}

// cmdGC handles the garbage collection command
func (dc *DebugConsole) cmdGC(args []string) string {
	before := runtime.NumGoroutine()
	runtime.GC()
	after := runtime.NumGoroutine()

	output := "🗑️  Garbage Collection:\n\n"
	output += fmt.Sprintf("Goroutines before: %d\n", before)
	output += fmt.Sprintf("Goroutines after:  %d\n", after)
	output += "✅ Garbage collection completed"

	return output
}

// cmdSummary handles the summary command
func (dc *DebugConsole) cmdSummary(args []string) string {
	summary := dc.debugManager.GetDebugSummary()

	output := "📋 Debug Summary:\n\n"

	// Format the summary nicely
	if flags, exists := summary["flags"]; exists {
		output += "🔧 Debug Flags:\n"
		flagsMap := flags.(DebugFlags)
		output += fmt.Sprintf("  Enabled: %t\n", flagsMap.Enabled)
		output += fmt.Sprintf("  Trace Active: %t\n", flagsMap.TraceActive)
		output += fmt.Sprintf("  State Inspection: %t\n", flagsMap.StateInspection)
		output += fmt.Sprintf("  Memory Inspection: %t\n", flagsMap.MemoryInspection)
		output += fmt.Sprintf("  Performance Profiling: %t\n", flagsMap.PerformanceProfiling)
		output += fmt.Sprintf("  Error Tracking: %t\n", flagsMap.ErrorTracking)
		output += fmt.Sprintf("  Progress Tracking: %t\n", flagsMap.ProgressTracking)
		output += "\n"
	}

	output += fmt.Sprintf("📊 Statistics:\n")
	output += fmt.Sprintf("  State Snapshots: %v\n", summary["state_snapshots"])
	output += fmt.Sprintf("  Memory Snapshots: %v\n", summary["memory_snapshots"])
	output += fmt.Sprintf("  Progress Trackers: %v\n", summary["progress_trackers"])
	output += fmt.Sprintf("  Goroutines: %v\n", summary["goroutines"])

	if currentMemory, exists := summary["current_memory"]; exists {
		memory := currentMemory.(map[string]interface{})
		output += fmt.Sprintf("\n💾 Current Memory:\n")
		output += fmt.Sprintf("  Alloc: %.2f MB\n", memory["alloc_mb"])
		output += fmt.Sprintf("  Total Alloc: %.2f MB\n", memory["total_alloc_mb"])
		output += fmt.Sprintf("  Sys: %.2f MB\n", memory["sys_mb"])
		output += fmt.Sprintf("  Heap Objects: %v\n", memory["heap_objects"])
	}

	return output
}

// ExecuteCommand executes a single debug command
func (dc *DebugConsole) ExecuteCommand(command string) string {
	commands := dc.getCommands()
	parts := strings.Fields(command)

	if len(parts) == 0 {
		return "❌ Empty command"
	}

	cmd := parts[0]
	args := parts[1:]

	return dc.executeCommand(commands, cmd, args)
}

// GetAvailableCommands returns list of available commands
func (dc *DebugConsole) GetAvailableCommands() []string {
	commands := dc.getCommands()
	cmdList := make([]string, 0, len(commands))

	for cmdName := range commands {
		cmdList = append(cmdList, cmdName)
	}

	return cmdList
}
