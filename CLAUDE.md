# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Run
- `go build` - Build the application
- `go run main.go` - Run the phone migration tool directly

### Testing
- `go test` - Run all tests (if any tests exist in the future)
- `go mod tidy` - Clean up and verify dependencies

### Environment Setup
- Set `ANTHROPIC_API_KEY` environment variable for Claude AI integration
- Example: `export ANTHROPIC_API_KEY=your_api_key_here`

### Git/GitHub Workflow

**GitHub CLI (gh) Commands:**
- `gh auth login` - Authenticate with GitHub
- `gh repo view` - View repository details
- `gh repo clone [repository]` - Clone a repository
- `gh pr create` - Create a new pull request
- `gh pr list` - List pull requests
- `gh pr view [PR number]` - View pull request details
- `gh pr checkout [PR number]` - Switch to a pull request branch
- `gh pr merge` - Merge a pull request
- `gh issue create` - Create a new issue
- `gh issue list` - List issues
- `gh release create` - Create a new release
- `gh release list` - List releases

## Architecture Overview

This is a Go-based phone system migration tool with Claude AI integration. The application uses the Bubble Tea TUI framework for an interactive command-line interface.

### Core Components

**Main Application Structure:**
- `main.go:1242-1247` - Entry point using Bubble Tea program
- `main.go:381-398` - Initial model setup with spinner and text input
- `main.go:404-599` - Main update loop handling state transitions and user input

**Data Models:**
- `TwilioPhoneSystem` and `TwilioUser` (lines 21-39) - Source format structures
- `RingCentralPhoneSystem` and `RingCentralAccount` (lines 41-59) - Target format structures  
- `ClaudeRequest/Response` (lines 62-86) - Claude API integration structures
- `MigrationPlan` and `TodoItem` (lines 94-108) - AI-generated migration planning

**Migration Engine:**
- `ClaudeEnhancedMigrator` (lines 89-92) - AI-powered migration orchestrator
- `PlanMigrationOrder()` (lines 279-360) - Claude analyzes data and creates step-by-step plans
- `AnalyzeDataQuality()` (lines 362-379) - Data validation and cleanup recommendations

**Conversion Logic:**
- `convertTwilioToRingCentral()` (lines 1151-1185) - Core format transformation
- `twilioToRingCentral()` and `ringCentralToTwilio()` - Bidirectional conversion functions
- Standard migration mode vs Claude-enhanced mode with AI planning

**UI States:**
- Progressive workflow: source → format → target → format → AI preference → plan review → execution
- Real-time step execution with visual progress indicators
- Styled output using Lipgloss for colors and formatting

### Key Features

**Claude AI Integration:**
- Intelligent migration order based on user roles and system impact
- Risk assessment and mitigation strategies  
- Step-by-step execution plans with detailed to-do lists
- Data quality analysis and cleanup recommendations

**Interactive TUI:**
- File selection and format specification
- Optional AI-enhanced planning workflow
- Real-time progress tracking during execution
- Comprehensive error handling and user feedback

**Sample Data:**
- `twilio-sample.json` - Example Twilio system export format
- `ringcentral-sample.json` - Target RingCentral format structure
- Converted output files demonstrate successful migrations

## Development Notes

**Dependencies:**
- Uses Bubble Tea (`github.com/charmbracelet/bubbletea`) for terminal UI
- Lipgloss for styling and layout
- Standard library for HTTP client and JSON processing
- Go 1.24.4 as specified in go.mod

**Claude API Integration:**
- Requires valid ANTHROPIC_API_KEY environment variable
- Uses Claude 3 Sonnet model for analysis and planning
- 30-second timeout for API calls
- Structured JSON prompts for consistent AI responses

**File Structure:**
- Single main.go file containing all functionality
- JSON sample files for testing different phone system formats
- No separate packages - everything in main package for simplicity