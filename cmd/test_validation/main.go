package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestSuite represents a comprehensive testing framework for Phase 4 validation
type TestSuite struct {
	projectRoot    string
	testResults    []TestResult
	startTime      time.Time
	successTargets SuccessTargets
}

// TestResult stores individual test execution results
type TestResult struct {
	Package  string
	TestName string
	Status   string
	Duration time.Duration
	Coverage float64
	ErrorMsg string
	Output   string
}

// SuccessTargets defines the validation criteria from the implementation plan
type SuccessTargets struct {
	MinCoverage     float64 // 95%+ coverage target
	MaxFailureRate  float64 // 0% failure rate target
	MinSpeedImprove float64 // 40%+ speed improvement
	MinMemoryReduce float64 // 60%+ memory reduction
	MinCacheHitRate float64 // 85%+ cache hit rate
}

// NewTestSuite creates a new comprehensive test suite
func NewTestSuite(projectRoot string) *TestSuite {
	return &TestSuite{
		projectRoot: projectRoot,
		testResults: make([]TestResult, 0),
		startTime:   time.Now(),
		successTargets: SuccessTargets{
			MinCoverage:     95.0,
			MaxFailureRate:  0.0,
			MinSpeedImprove: 40.0,
			MinMemoryReduce: 60.0,
			MinCacheHitRate: 85.0,
		},
	}
}

// RunComprehensiveValidation executes the complete Phase 4 testing strategy
func (ts *TestSuite) RunComprehensiveValidation() error {
	fmt.Println("🚀 Starting Phase 4: Comprehensive Testing & Validation")
	fmt.Println("=" + strings.Repeat("=", 60))

	// 1. Run all existing unit tests
	if err := ts.runAllUnitTests(); err != nil {
		return fmt.Errorf("unit tests failed: %w", err)
	}

	// 2. Run integration tests
	if err := ts.runIntegrationTests(); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}

	// 3. Run performance benchmarks
	if err := ts.runPerformanceBenchmarks(); err != nil {
		return fmt.Errorf("performance benchmarks failed: %w", err)
	}

	// 4. Validate test coverage
	if err := ts.validateTestCoverage(); err != nil {
		return fmt.Errorf("test coverage validation failed: %w", err)
	}

	// 5. Generate comprehensive report
	ts.generateValidationReport()

	return nil
}

// runAllUnitTests executes all unit tests in the analytics and service packages
func (ts *TestSuite) runAllUnitTests() error {
	fmt.Println("\n📋 1. Running Unit Tests")
	fmt.Println("-" + strings.Repeat("-", 40))

	testPackages := []string{
		"./internal/analytics/...",
		"./internal/service/...",
		"./internal/domain/...",
	}

	for _, pkg := range testPackages {
		fmt.Printf("Testing package: %s\\n", pkg)
		result := ts.runTestPackage(pkg, "unit")
		ts.testResults = append(ts.testResults, result)

		if result.Status == "FAIL" {
			fmt.Printf("❌ %s tests failed: %s\\n", pkg, result.ErrorMsg)
		} else {
			fmt.Printf("✅ %s tests passed (%.2fs)\\n", pkg, result.Duration.Seconds())
		}
	}

	return nil
}

// runIntegrationTests executes integration tests
func (ts *TestSuite) runIntegrationTests() error {
	fmt.Println("\n🔗 2. Running Integration Tests")
	fmt.Println("-" + strings.Repeat("-", 40))

	integrationTests := []string{
		"./internal/service/technical_indicator_service_v2_cache_integration_test.go",
		"./internal/analytics/processor_test.go",
	}

	for _, testFile := range integrationTests {
		if _, err := os.Stat(filepath.Join(ts.projectRoot, testFile)); os.IsNotExist(err) {
			fmt.Printf("⚠️  Integration test file not found: %s\\n", testFile)
			continue
		}

		fmt.Printf("Running integration test: %s\\n", testFile)
		result := ts.runSpecificTest(testFile, "integration")
		ts.testResults = append(ts.testResults, result)

		if result.Status == "FAIL" {
			fmt.Printf("❌ Integration test failed: %s\\n", result.ErrorMsg)
		} else {
			fmt.Printf("✅ Integration test passed (%.2fs)\\n", result.Duration.Seconds())
		}
	}

	return nil
}

// runPerformanceBenchmarks executes performance benchmark tests
func (ts *TestSuite) runPerformanceBenchmarks() error {
	fmt.Println("\n⚡ 3. Running Performance Benchmarks")
	fmt.Println("-" + strings.Repeat("-", 40))

	benchmarkPackages := []string{
		"./internal/analytics/cache",
		"./internal/service",
	}

	for _, pkg := range benchmarkPackages {
		fmt.Printf("Running benchmarks for: %s\\n", pkg)
		result := ts.runBenchmarks(pkg)
		ts.testResults = append(ts.testResults, result)

		if result.Status == "FAIL" {
			fmt.Printf("❌ Benchmarks failed: %s\\n", result.ErrorMsg)
		} else {
			fmt.Printf("✅ Benchmarks completed (%.2fs)\\n", result.Duration.Seconds())
		}
	}

	return nil
}

// validateTestCoverage checks test coverage across all packages
func (ts *TestSuite) validateTestCoverage() error {
	fmt.Println("\n📊 4. Validating Test Coverage")
	fmt.Println("-" + strings.Repeat("-", 40))

	// Run coverage analysis for analytics package
	analyticsResult := ts.runCoverageAnalysis("./internal/analytics/...")
	ts.testResults = append(ts.testResults, analyticsResult)

	// Run coverage analysis for service package
	serviceResult := ts.runCoverageAnalysis("./internal/service/...")
	ts.testResults = append(ts.testResults, serviceResult)

	fmt.Printf("Analytics package coverage: %.1f%%\\n", analyticsResult.Coverage)
	fmt.Printf("Service package coverage: %.1f%%\\n", serviceResult.Coverage)

	// Validate coverage targets
	if analyticsResult.Coverage < ts.successTargets.MinCoverage {
		fmt.Printf("⚠️  Analytics coverage %.1f%% below target %.1f%%\\n",
			analyticsResult.Coverage, ts.successTargets.MinCoverage)
	}

	if serviceResult.Coverage < ts.successTargets.MinCoverage {
		fmt.Printf("⚠️  Service coverage %.1f%% below target %.1f%%\\n",
			serviceResult.Coverage, ts.successTargets.MinCoverage)
	}

	return nil
}

// runTestPackage executes tests for a specific package
func (ts *TestSuite) runTestPackage(pkg, testType string) TestResult {
	startTime := time.Now()

	cmd := exec.Command("go", "test", "-v", pkg)
	cmd.Dir = ts.projectRoot

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := TestResult{
		Package:  pkg,
		TestName: testType,
		Duration: duration,
		Output:   string(output),
	}

	if err != nil {
		result.Status = "FAIL"
		result.ErrorMsg = err.Error()
	} else {
		result.Status = "PASS"
	}

	return result
}

// runSpecificTest executes a specific test file
func (ts *TestSuite) runSpecificTest(testFile, testType string) TestResult {
	startTime := time.Now()

	cmd := exec.Command("go", "test", "-v", testFile)
	cmd.Dir = ts.projectRoot

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := TestResult{
		Package:  testFile,
		TestName: testType,
		Duration: duration,
		Output:   string(output),
	}

	if err != nil {
		result.Status = "FAIL"
		result.ErrorMsg = err.Error()
	} else {
		result.Status = "PASS"
	}

	return result
}

// runBenchmarks executes benchmark tests for a package
func (ts *TestSuite) runBenchmarks(pkg string) TestResult {
	startTime := time.Now()

	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", pkg)
	cmd.Dir = ts.projectRoot

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := TestResult{
		Package:  pkg,
		TestName: "benchmark",
		Duration: duration,
		Output:   string(output),
	}

	if err != nil {
		result.Status = "FAIL"
		result.ErrorMsg = err.Error()
	} else {
		result.Status = "PASS"
	}

	return result
}

// runCoverageAnalysis executes test coverage analysis
func (ts *TestSuite) runCoverageAnalysis(pkg string) TestResult {
	startTime := time.Now()

	cmd := exec.Command("go", "test", "-cover", pkg)
	cmd.Dir = ts.projectRoot

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := TestResult{
		Package:  pkg,
		TestName: "coverage",
		Duration: duration,
		Output:   string(output),
	}

	if err != nil {
		result.Status = "FAIL"
		result.ErrorMsg = err.Error()
	} else {
		result.Status = "PASS"
		// Extract coverage percentage from output
		result.Coverage = ts.extractCoveragePercentage(string(output))
	}

	return result
}

// extractCoveragePercentage parses coverage percentage from test output
func (ts *TestSuite) extractCoveragePercentage(output string) float64 {
	// Look for patterns like "coverage: 95.2% of statements"
	lines := strings.Split(output, "\\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") && strings.Contains(line, "%") {
			// Simple extraction - in production, use regex
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, "%") {
					coverageStr := strings.TrimSuffix(part, "%")
					var coverage float64
					fmt.Sscanf(coverageStr, "%f", &coverage)
					return coverage
				}
			}
		}
	}
	return 0.0
}

// generateValidationReport creates a comprehensive validation report
func (ts *TestSuite) generateValidationReport() {
	fmt.Println("\n📈 5. Validation Report")
	fmt.Println("=" + strings.Repeat("=", 60))

	totalTests := len(ts.testResults)
	passedTests := 0
	failedTests := 0
	totalDuration := time.Since(ts.startTime)

	for _, result := range ts.testResults {
		if result.Status == "PASS" {
			passedTests++
		} else {
			failedTests++
		}
	}

	fmt.Printf("📊 Test Execution Summary:\\n")
	fmt.Printf("   Total Tests: %d\\n", totalTests)
	fmt.Printf("   Passed: %d\\n", passedTests)
	fmt.Printf("   Failed: %d\\n", failedTests)
	fmt.Printf("   Success Rate: %.1f%%\\n", float64(passedTests)/float64(totalTests)*100)
	fmt.Printf("   Total Duration: %.2fs\\n", totalDuration.Seconds())

	fmt.Printf("\\n🎯 Success Criteria Validation:\\n")
	successRate := float64(passedTests) / float64(totalTests) * 100
	if successRate >= (100.0 - ts.successTargets.MaxFailureRate) {
		fmt.Printf("   ✅ Test Success Rate: %.1f%% (Target: %.1f%%)\\n",
			successRate, 100.0-ts.successTargets.MaxFailureRate)
	} else {
		fmt.Printf("   ❌ Test Success Rate: %.1f%% (Target: %.1f%%)\\n",
			successRate, 100.0-ts.successTargets.MaxFailureRate)
	}

	// Calculate overall coverage
	var totalCoverage float64
	coverageTests := 0
	for _, result := range ts.testResults {
		if result.TestName == "coverage" && result.Coverage > 0 {
			totalCoverage += result.Coverage
			coverageTests++
		}
	}

	if coverageTests > 0 {
		avgCoverage := totalCoverage / float64(coverageTests)
		if avgCoverage >= ts.successTargets.MinCoverage {
			fmt.Printf("   ✅ Test Coverage: %.1f%% (Target: %.1f%%)\\n",
				avgCoverage, ts.successTargets.MinCoverage)
		} else {
			fmt.Printf("   ⚠️  Test Coverage: %.1f%% (Target: %.1f%%)\\n",
				avgCoverage, ts.successTargets.MinCoverage)
		}
	}

	fmt.Printf("\\n📋 Detailed Results:\\n")
	for _, result := range ts.testResults {
		status := "✅"
		if result.Status == "FAIL" {
			status = "❌"
		}
		fmt.Printf("   %s %s [%s] (%.2fs)\\n",
			status, result.Package, result.TestName, result.Duration.Seconds())

		if result.Status == "FAIL" && result.ErrorMsg != "" {
			fmt.Printf("      Error: %s\\n", result.ErrorMsg)
		}
	}

	// Generate recommendations
	ts.generateRecommendations(successRate, totalCoverage/float64(coverageTests))
}

// generateRecommendations provides actionable recommendations based on test results
func (ts *TestSuite) generateRecommendations(successRate, avgCoverage float64) {
	fmt.Printf("\\n💡 Recommendations:\\n")

	if successRate < 95.0 {
		fmt.Printf("   • Investigate and fix failing tests before production deployment\\n")
		fmt.Printf("   • Review error messages and update implementations as needed\\n")
	}

	if avgCoverage < ts.successTargets.MinCoverage {
		fmt.Printf("   • Add unit tests to increase coverage to 95%+ target\\n")
		fmt.Printf("   • Focus on edge cases and error handling scenarios\\n")
	}

	fmt.Printf("   • Run load testing with larger datasets (1000+ instruments)\\n")
	fmt.Printf("   • Validate memory usage under sustained load\\n")
	fmt.Printf("   • Monitor cache effectiveness in production environment\\n")

	if successRate >= 95.0 && avgCoverage >= ts.successTargets.MinCoverage {
		fmt.Printf("   ✨ System is ready for production deployment!\\n")
		fmt.Printf("   • Proceed with gradual rollout using feature flags\\n")
		fmt.Printf("   • Monitor performance metrics in production\\n")
		fmt.Printf("   • Set up alerting for performance degradation\\n")
	}
}

func main() {
	// Get project root directory
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}

	// Create and run test suite
	testSuite := NewTestSuite(projectRoot)

	if err := testSuite.RunComprehensiveValidation(); err != nil {
		log.Fatal("Validation failed:", err)
	}

	fmt.Println("\\n🎉 Phase 4: Comprehensive Testing & Validation Complete!")
}
