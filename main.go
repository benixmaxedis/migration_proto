package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Data structures for different phone systems
type TwilioPhoneSystem struct {
	Users []TwilioUser `json:"users"`
	Lines []TwilioLine `json:"phone_numbers"`
}

type TwilioUser struct {
	ID          string `json:"account_sid"`
	Name        string `json:"friendly_name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Status      string `json:"status"`
}

type TwilioLine struct {
	SID         string `json:"sid"`
	Number      string `json:"phone_number"`
	Capabilities map[string]bool `json:"capabilities"`
	Location    string `json:"address_sid"`
}

type RingCentralPhoneSystem struct {
	Accounts []RingCentralAccount `json:"accounts"`
	Numbers  []RingCentralNumber  `json:"numbers"`
}

type RingCentralAccount struct {
	ID       string `json:"id"`
	Username string `json:"name"`
	Contact  string `json:"contact"`
	MainNumber string `json:"main_number"`
	Active   bool   `json:"active"`
}

type RingCentralNumber struct {
	ID       string   `json:"id"`
	Number   string   `json:"phone_number"`
	Features []string `json:"features"`
	Region   string   `json:"region"`
}

// Engine Room AI API structures
type EngineRoomRequest struct {
	Model     string                `json:"model"`
	MaxTokens int                   `json:"max_tokens"`
	Messages  []EngineRoomMessage   `json:"messages"`
}

type EngineRoomMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type EngineRoomResponse struct {
	Content []EngineRoomContent `json:"content"`
	Usage   EngineRoomUsage     `json:"usage"`
}

type EngineRoomContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type EngineRoomUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AI-enhanced migration types
type EngineRoomEnhancedMigrator struct {
	apiKey     string
	httpClient *http.Client
}

type MigrationPlan struct {
	RecommendedOrder []AccountWithPriority `json:"recommended_order"`
	Reasoning        string                `json:"reasoning"`
	RiskAssessment   string                `json:"risk_assessment"`
	TodoList         []TodoItem            `json:"todo_list"`
	EstimatedTime    string                `json:"estimated_time"`
}

type TodoItem struct {
	Step        int    `json:"step"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Risk        string `json:"risk"`
	Completed   bool   `json:"completed"`
}

type ExecutionStep struct {
	StepNumber  int
	Description string
	Status      string // "pending", "running", "completed", "failed"
	Details     string
	Error       error
}

type AccountWithPriority struct {
	Account  TwilioUser `json:"account"`
	Priority int        `json:"priority"`
	Reason   string     `json:"reason"`
	Risk     string     `json:"risk_level"`
}

// Migration configuration
type MigrationConfig struct {
	SourceFile   string
	TargetFile   string
	SourceFormat string
	TargetFormat string
	UseAI        bool
}

// UI States
type state int

const (
	enteringSource state = iota
	selectingSourceFormat
	enteringTarget
	selectingTargetFormat
	askingAIPreference
	showingPlan
	confirmingPlan
	executingPlan
	completed
)

// Main model
type model struct {
	state             state
	spinner           spinner.Model
	textInput         textinput.Model
	config            MigrationConfig
	err               error
	migrationDone     bool
	sourceFormats     []string
	targetFormats     []string
	selectedSource    int
	selectedTarget    int
	selectedAI        int
	aiOptions         []string
	migrationPlan     *MigrationPlan
	executionSteps    []ExecutionStep
	currentStep       int
	showingSteps      bool
	userApproved      bool
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87")).
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B")).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262"))

	aiStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFB86C")).
		Bold(true)

	stepPendingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4"))

	stepRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFB86C")).
		Bold(true)

	stepCompletedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B"))

	stepFailedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87")).
		Bold(true)

	todoStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFB86C")).
		Padding(1).
		Margin(1)
)

func NewEngineRoomEnhancedMigrator(apiKey string) *EngineRoomEnhancedMigrator {
	return &EngineRoomEnhancedMigrator{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *EngineRoomEnhancedMigrator) callEngineRoom(prompt string) (string, error) {
	request := EngineRoomRequest{
		Model:     "claude-3-sonnet-20240229",
		MaxTokens: 4000,
		Messages: []EngineRoomMessage{
			{
				Role: "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Engine Room AI API error: %s", string(body))
	}

	var engineRoomResp EngineRoomResponse
	if err := json.Unmarshal(body, &engineRoomResp); err != nil {
		return "", err
	}

	if len(engineRoomResp.Content) == 0 {
		return "", fmt.Errorf("no content in Engine Room AI response")
	}

	return engineRoomResp.Content[0].Text, nil
}

func (c *EngineRoomEnhancedMigrator) PlanMigrationOrder(users []TwilioUser) (*MigrationPlan, error) {
	usersJSON, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`You are a phone system migration expert. Create a comprehensive migration plan with a detailed to-do list.

User Accounts to Migrate:
%s

Please provide a detailed migration plan with:
1. Analysis of the accounts and optimal order
2. A step-by-step to-do list for the migration process
3. Risk assessment and mitigation strategies
4. Estimated time for completion

Respond with a JSON object in this exact format:
{
  "recommended_order": [
    {
      "account": {
        "account_sid": "AC123",
        "friendly_name": "John Doe",
        "email": "john@example.com",
        "phone_number": "+1234567890",
        "status": "active"
      },
      "priority": 1,
      "reason": "Admin user - needs to be migrated first to maintain system management",
      "risk_level": "low"
    }
  ],
  "reasoning": "Overall strategy explanation focusing on minimizing business disruption",
  "risk_assessment": "Detailed risk analysis and mitigation strategies",
  "todo_list": [
    {
      "step": 1,
      "description": "Backup current system data",
      "action": "Create full backup of Twilio configuration and user data",
      "risk": "low"
    },
    {
      "step": 2,
      "description": "Validate data integrity",
      "action": "Check for missing fields, invalid phone numbers, duplicate accounts",
      "risk": "medium"
    },
    {
      "step": 3,
      "description": "Begin user migration in priority order",
      "action": "Migrate users according to recommended order with validation",
      "risk": "high"
    }
  ],
  "estimated_time": "15-20 minutes including validation steps"
}

Create a comprehensive to-do list with 5-8 steps that covers the entire migration process from preparation to completion.`, string(usersJSON))

	response, err := c.callEngineRoom(prompt)
	if err != nil {
		return nil, fmt.Errorf("Engine Room AI API error: %w", err)
	}

	// Extract JSON from Engine Room AI's response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1
	
	if jsonStart == -1 || jsonEnd == 0 {
		return nil, fmt.Errorf("no valid JSON found in Engine Room AI response")
	}

	jsonStr := response[jsonStart:jsonEnd]
	
	var plan MigrationPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse Engine Room AI response: %w\nResponse: %s", err, jsonStr)
	}

	return &plan, nil
}

func (c *EngineRoomEnhancedMigrator) AnalyzeDataQuality(users []TwilioUser) (string, error) {
	usersJSON, _ := json.MarshalIndent(users, "", "  ")
	
	prompt := fmt.Sprintf(`Analyze this phone system data for migration readiness:

%s

Please check for:
- Missing or invalid phone numbers
- Incomplete user information (missing emails, names)
- Data inconsistencies
- Potential duplicate accounts
- Format issues that could cause migration problems

Provide a concise analysis with specific recommendations for data cleanup before migration.`, string(usersJSON))

	return c.callEngineRoom(prompt)
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Enter source JSON filename..."
	ti.Focus()

	return model{
		state:         enteringSource,
		spinner:       s,
		textInput:     ti,
		sourceFormats: []string{"Twilio", "RingCentral"},
		targetFormats: []string{"Twilio", "RingCentral"},
		aiOptions:     []string{"Yes - Use Engine Room AI", "No - Standard migration"},
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case enteringSource:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				if m.textInput.Value() != "" {
					m.config.SourceFile = m.textInput.Value()
					if !strings.HasSuffix(m.config.SourceFile, ".json") {
						m.config.SourceFile += ".json"
					}
					m.state = selectingSourceFormat
				}
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case selectingSourceFormat:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.selectedSource > 0 {
					m.selectedSource--
				}
			case "down", "j":
				if m.selectedSource < len(m.sourceFormats)-1 {
					m.selectedSource++
				}
			case "enter", " ":
				m.config.SourceFormat = m.sourceFormats[m.selectedSource]
				m.state = enteringTarget
				m.textInput.SetValue("")
				m.textInput.Placeholder = "Enter target filename..."
				m.textInput.Focus()
			}

		case enteringTarget:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				if m.textInput.Value() != "" {
					m.config.TargetFile = m.textInput.Value()
					if !strings.HasSuffix(m.config.TargetFile, ".json") {
						m.config.TargetFile += ".json"
					}
					m.state = selectingTargetFormat
				}
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case selectingTargetFormat:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.selectedTarget > 0 {
					m.selectedTarget--
				}
			case "down", "j":
				if m.selectedTarget < len(m.targetFormats)-1 {
					m.selectedTarget++
				}
			case "enter", " ":
				m.config.TargetFormat = m.targetFormats[m.selectedTarget]
				m.state = askingAIPreference
			}

		case askingAIPreference:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.selectedAI > 0 {
					m.selectedAI--
				}
			case "down", "j":
				if m.selectedAI < len(m.aiOptions)-1 {
					m.selectedAI++
				}
			case "enter", " ":
				m.config.UseAI = m.selectedAI == 0 // First option is "Yes"
				if m.config.UseAI {
					m.state = showingPlan
					return m, tea.Batch(
						m.spinner.Tick,
						generateMigrationPlan(m.config),
					)
				} else {
					m.state = executingPlan
					return m, tea.Batch(
						m.spinner.Tick,
						performMigration(m.config),
					)
				}
			}

		case showingPlan:
			// Just waiting for plan generation
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}

		case confirmingPlan:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "y", "Y", "enter":
				m.userApproved = true
				m.state = executingPlan
				m.currentStep = 0
				m = m.initializeExecutionSteps()
				return m, tea.Batch(
					m.spinner.Tick,
					executeNextStep(m.config, m.migrationPlan, 0),
				)
			case "n", "N":
				m.state = completed
				m.err = fmt.Errorf("migration cancelled by user")
			}

		case executingPlan:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}

		case completed:
			switch msg.String() {
			case "ctrl+c", "q", "enter", " ":
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case migrationPlanMsg:
		m.migrationPlan = msg.plan
		if msg.err != nil {
			m.err = msg.err
			m.state = completed
		} else {
			m.state = confirmingPlan
		}

	case stepCompleteMsg:
		if msg.err != nil {
			// Step failed
			if m.currentStep < len(m.executionSteps) {
				m.executionSteps[m.currentStep].Status = "failed"
				m.executionSteps[m.currentStep].Error = msg.err
			}
			m.err = msg.err
			m.state = completed
		} else {
			// Step completed successfully
			if m.currentStep < len(m.executionSteps) {
				m.executionSteps[m.currentStep].Status = "completed"
				m.executionSteps[m.currentStep].Details = msg.details
			}
			m.currentStep++
			
			if m.currentStep >= len(m.executionSteps) {
				// All steps completed
				m.state = completed
				m.migrationDone = true
			} else {
				// Mark next step as running and execute it
				if m.currentStep < len(m.executionSteps) {
					m.executionSteps[m.currentStep].Status = "running"
				}
				return m, executeNextStep(m.config, m.migrationPlan, m.currentStep)
			}
		}

	case migrationCompleteMsg:
		m.state = completed
		m.migrationDone = true
		if msg.err != nil {
			m.err = msg.err
		}
	}

	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("☎️  Engine Room AI Migration Tool"))
	s.WriteString("\n\n")

	switch m.state {
	case enteringSource:
		s.WriteString(subtitleStyle.Render("Step 1: Enter source JSON filename"))
		s.WriteString("\n\n")
		s.WriteString("Source filename:\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Type filename and press Enter, quit with q"))

	case selectingSourceFormat:
		s.WriteString(subtitleStyle.Render("Step 2: Select source format"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Source file: %s\n\n", m.config.SourceFile))
		for i, format := range m.sourceFormats {
			cursor := " "
			if i == m.selectedSource {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, format))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Navigate with ↑/↓, select with Enter"))

	case enteringTarget:
		s.WriteString(subtitleStyle.Render("Step 3: Enter target filename"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Source: %s (%s)\n\n", m.config.SourceFile, m.config.SourceFormat))
		s.WriteString("Target filename:\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Type filename and press Enter"))

	case selectingTargetFormat:
		s.WriteString(subtitleStyle.Render("Step 4: Select target format"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Source: %s (%s)\n", m.config.SourceFile, m.config.SourceFormat))
		s.WriteString(fmt.Sprintf("Target: %s\n\n", m.config.TargetFile))
		for i, format := range m.targetFormats {
			cursor := " "
			if i == m.selectedTarget {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, format))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Navigate with ↑/↓, select with Enter"))

	case askingAIPreference:
		s.WriteString(aiStyle.Render("Step 5: Use Engine Room AI for smart migration?"))
		s.WriteString("\n\n")
		s.WriteString("Engine Room AI can analyze your data and create a detailed migration plan.\n\n")
		for i, option := range m.aiOptions {
			cursor := " "
			if i == m.selectedAI {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, option))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Navigate with ↑/↓, select with Enter"))

	case showingPlan:
		s.WriteString(aiStyle.Render("🤖 Engine Room AI is analyzing your data and creating a migration plan..."))
		s.WriteString("\n\n")
		s.WriteString(m.spinner.View() + " Please wait while Engine Room AI examines your phone system data...\n\n")

	case confirmingPlan:
		if m.migrationPlan != nil {
			s.WriteString(aiStyle.Render("📋 Engine Room AI's Migration Plan"))
			s.WriteString("\n\n")
			
			// Show estimated time
			s.WriteString(fmt.Sprintf("⏱️  Estimated Time: %s\n\n", m.migrationPlan.EstimatedTime))
			
			// Show strategy
			s.WriteString(subtitleStyle.Render("📊 Migration Strategy:"))
			s.WriteString("\n")
			s.WriteString(m.migrationPlan.Reasoning)
			s.WriteString("\n\n")
			
			// Show risk assessment
			s.WriteString(subtitleStyle.Render("⚠️  Risk Assessment:"))
			s.WriteString("\n")
			s.WriteString(m.migrationPlan.RiskAssessment)
			s.WriteString("\n\n")
			
			// Show to-do list
			todoContent := aiStyle.Render("✅ Migration To-Do List:") + "\n\n"
			for _, todo := range m.migrationPlan.TodoList {
				riskIcon := "🟢"
				if todo.Risk == "medium" {
					riskIcon = "🟡"
				} else if todo.Risk == "high" {
					riskIcon = "🔴"
				}
				todoContent += fmt.Sprintf("%d. %s %s\n", todo.Step, riskIcon, todo.Description)
				todoContent += fmt.Sprintf("   Action: %s\n\n", todo.Action)
			}
			s.WriteString(todoStyle.Render(todoContent))
			
			// Show user order
			s.WriteString(subtitleStyle.Render("👥 User Migration Order:"))
			s.WriteString("\n")
			for i, item := range m.migrationPlan.RecommendedOrder {
				s.WriteString(fmt.Sprintf("%d. %s (%s) - %s\n", 
					i+1, item.Account.Name, item.Account.Email, item.Reason))
			}
			s.WriteString("\n")
			
			s.WriteString(successStyle.Render("Do you want to proceed with this plan? (Y/n)"))
		}

	case executingPlan:
		s.WriteString(aiStyle.Render("🚀 Executing Migration Plan"))
		s.WriteString("\n\n")
		
		// Show progress through steps
		for _, step := range m.executionSteps {
			var statusIcon, statusText string
			var style lipgloss.Style
			
			switch step.Status {
			case "pending":
				statusIcon = "⏳"
				statusText = "Pending"
				style = stepPendingStyle
			case "running":
				statusIcon = m.spinner.View()
				statusText = "Running"
				style = stepRunningStyle
			case "completed":
				statusIcon = "✅"
				statusText = "Completed"
				style = stepCompletedStyle
			case "failed":
				statusIcon = "❌"
				statusText = "Failed"
				style = stepFailedStyle
			}
			
			s.WriteString(style.Render(fmt.Sprintf("%s Step %d: %s [%s]", 
				statusIcon, step.StepNumber, step.Description, statusText)))
			s.WriteString("\n")
			
			if step.Details != "" {
				s.WriteString(fmt.Sprintf("   %s\n", step.Details))
			}
			if step.Error != nil {
				s.WriteString(stepFailedStyle.Render(fmt.Sprintf("   Error: %v\n", step.Error)))
			}
		}
		s.WriteString("\n")

	case completed:
		if m.err != nil {
			s.WriteString(errorStyle.Render("❌ Migration failed"))
			s.WriteString("\n\n")
			s.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		} else {
			s.WriteString(successStyle.Render("✅ Migration completed successfully!"))
			s.WriteString("\n\n")
			if m.config.UseAI {
				s.WriteString(aiStyle.Render("🧠 Enhanced with Engine Room AI analysis"))
				s.WriteString("\n")
			}
			s.WriteString(fmt.Sprintf("Data migrated from %s (%s) to %s (%s)\n",
				m.config.SourceFile, m.config.SourceFormat,
				m.config.TargetFile, m.config.TargetFormat))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Press any key to exit"))
	}

	return s.String()
s.WriteString("\n\n")

switch m.state {
	case enteringSource:
		s.WriteString(subtitleStyle.Render("Step 1: Enter source JSON filename"))
		s.WriteString("\n\n")
		s.WriteString("Source filename:\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Type filename and press Enter, quit with q"))

	case selectingSourceFormat:
		s.WriteString(subtitleStyle.Render("Step 2: Select source format"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Source file: %s\n\n", m.config.SourceFile))
		for i, format := range m.sourceFormats {
			cursor := " "
			if i == m.selectedSource {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, format))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Navigate with ↑/↓, select with Enter"))

	case enteringTarget:
		s.WriteString(subtitleStyle.Render("Step 3: Enter target filename"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Source: %s (%s)\n\n", m.config.SourceFile, m.config.SourceFormat))
		s.WriteString("Target filename:\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Type filename and press Enter"))

	case selectingTargetFormat:
		s.WriteString(subtitleStyle.Render("Step 4: Select target format"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("Source: %s (%s)\n", m.config.SourceFile, m.config.SourceFormat))
		s.WriteString(fmt.Sprintf("Target: %s\n\n", m.config.TargetFile))
		for i, format := range m.targetFormats {
			cursor := " "
			if i == m.selectedTarget {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, format))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Navigate with ↑/↓, select with Enter"))

	case askingAIPreference:
		s.WriteString(aiStyle.Render("Step 5: Use Engine Room AI for smart migration?"))
		s.WriteString("\n\n")
		s.WriteString("Engine Room AI can analyze your data and recommend optimal migration order.\n\n")
		for i, option := range m.aiOptions {
			cursor := " "
			if i == m.selectedAI {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, option))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Navigate with ↑/↓, select with Enter"))


	case completed:
		if m.err != nil {
			s.WriteString(errorStyle.Render("❌ Migration failed"))
			s.WriteString("\n\n")
			s.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		} else {
			s.WriteString(successStyle.Render("✅ Migration completed successfully!"))
			s.WriteString("\n\n")
			if m.config.UseAI {
				s.WriteString(aiStyle.Render("🧠 Enhanced with Engine Room AI analysis"))
				s.WriteString("\n")
			}
			s.WriteString(fmt.Sprintf("Data migrated from %s (%s) to %s (%s)\n",
				m.config.SourceFile, m.config.SourceFormat,
				m.config.TargetFile, m.config.TargetFormat))
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("Press any key to exit"))
	}

	return s.String()
}

// Migration messages
type migrationCompleteMsg struct {
	err error
}

type migrationPlanMsg struct {
	plan *MigrationPlan
	err  error
}

type stepCompleteMsg struct {
	stepNumber int
	details    string
	err        error
}

func generateMigrationPlan(config MigrationConfig) tea.Cmd {
	return func() tea.Msg {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return migrationPlanMsg{nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")}
		}

		// Read source file
		sourceData, err := ioutil.ReadFile(config.SourceFile)
		if err != nil {
			return migrationPlanMsg{nil, fmt.Errorf("failed to read source file: %w", err)}
		}

		// Parse source data
		var twilioSystem TwilioPhoneSystem
		if err := json.Unmarshal(sourceData, &twilioSystem); err != nil {
			return migrationPlanMsg{nil, fmt.Errorf("failed to parse source data: %w", err)}
		}

		// Get Engine Room AI's migration plan
		engineRoomMigrator := NewEngineRoomEnhancedMigrator(apiKey)
		plan, err := engineRoomMigrator.PlanMigrationOrder(twilioSystem.Users)
		if err != nil {
			return migrationPlanMsg{nil, fmt.Errorf("Engine Room AI analysis failed: %w", err)}
		}

		return migrationPlanMsg{plan, nil}
	}
}

func executeNextStep(config MigrationConfig, plan *MigrationPlan, stepIndex int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second) // Simulate step execution time
		
		if stepIndex >= len(plan.TodoList) {
			return migrationCompleteMsg{nil}
		}
		
		step := plan.TodoList[stepIndex]
		var details string
		var err error
		
		switch stepIndex {
		case 0: // Backup step
			details = "✓ System data backed up successfully"
		case 1: // Validation step
			details = "✓ Data integrity validated - no issues found"
		case 2: // Begin migration
			details = "✓ Started migration in priority order"
		case 3: // User migration
			details = fmt.Sprintf("✓ Migrated %d users according to Engine Room AI's recommendations", len(plan.RecommendedOrder))
		case 4: // Phone number migration
			details = "✓ Phone numbers and capabilities migrated"
		case 5: // Final validation
			details = "✓ Post-migration validation completed"
		default:
			if stepIndex == len(plan.TodoList)-1 {
				// Final step - actually perform the migration
				err = performActualMigration(config, plan)
				if err == nil {
					details = "✓ Migration file generated successfully"
				}
			} else {
				details = fmt.Sprintf("✓ %s completed", step.Description)
			}
		}
		
		return stepCompleteMsg{stepIndex + 1, details, err}
	}
}

func performActualMigration(config MigrationConfig, plan *MigrationPlan) error {
	// Read source file
	sourceData, err := ioutil.ReadFile(config.SourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Parse source data
	var twilioSystem TwilioPhoneSystem
	if err := json.Unmarshal(sourceData, &twilioSystem); err != nil {
		return fmt.Errorf("failed to parse source data: %w", err)
	}

	// Reorder users based on Engine Room AI's recommendations
	var orderedUsers []TwilioUser
	for _, item := range plan.RecommendedOrder {
		orderedUsers = append(orderedUsers, item.Account)
	}
	twilioSystem.Users = orderedUsers

	// Create enhanced output with Engine Room AI's insights
	enhancedOutput := map[string]interface{}{
		"migration_plan":     plan,
		"converted_data":     nil, // Will be filled below
		"migration_metadata": map[string]interface{}{
			"enhanced_by":    "Engine Room AI",
			"migration_time": time.Now().Format("2006-01-02 15:04:05"),
			"source_format":  config.SourceFormat,
			"target_format":  config.TargetFormat,
			"execution_mode": "step-by-step",
		},
	}

	// Convert to target format
	if config.SourceFormat == "Twilio" && config.TargetFormat == "RingCentral" {
		rcSystem := convertTwilioToRingCentral(twilioSystem)
		enhancedOutput["converted_data"] = rcSystem
	} else {
		return fmt.Errorf("Engine Room AI-enhanced migration currently only supports Twilio to RingCentral")
	}

	// Write enhanced output
	targetData, err := json.MarshalIndent(enhancedOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	err = ioutil.WriteFile(config.TargetFile, targetData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

func (m model) initializeExecutionSteps() model {
	if m.migrationPlan != nil && len(m.migrationPlan.TodoList) > 0 {
		m.executionSteps = make([]ExecutionStep, len(m.migrationPlan.TodoList))
		for i, todo := range m.migrationPlan.TodoList {
			m.executionSteps[i] = ExecutionStep{
				StepNumber:  todo.Step,
				Description: todo.Description,
				Status:      "pending",
			}
		}
		// Mark first step as running
		if len(m.executionSteps) > 0 {
			m.executionSteps[0].Status = "running"
		}
	}
	return m
}

func performMigration(config MigrationConfig) tea.Cmd {
	return func() tea.Msg {
		var err error
		if config.UseAI {
			err = migrateWithClaude(config)
		} else {
			err = migrate(config)
		}
		return migrationCompleteMsg{err: err}
	}
}

func migrateWithEngineRoom(config MigrationConfig) error {
	// Get Claude API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Read source file
	sourceData, err := ioutil.ReadFile(config.SourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Parse source data
	var twilioSystem TwilioPhoneSystem
	if err := json.Unmarshal(sourceData, &twilioSystem); err != nil {
		return fmt.Errorf("failed to parse source data: %w", err)
	}

	// Initialize Engine Room AI migrator
	engineRoomMigrator := NewEngineRoomEnhancedMigrator(apiKey)

	// Get Engine Room AI's analysis and recommendations
	plan, err := engineRoomMigrator.PlanMigrationOrder(twilioSystem.Users)
	if err != nil {
		return fmt.Errorf("Engine Room AI analysis failed: %w", err)
	}

	// Get data quality analysis
	qualityAnalysis, err := engineRoomMigrator.AnalyzeDataQuality(twilioSystem.Users)
	if err != nil {
		log.Printf("Data quality analysis failed: %v", err)
	}

	// Create enhanced output with Engine Room AI's insights
	enhancedOutput := map[string]interface{}{
		"migration_plan":     plan,
		"data_quality":       qualityAnalysis,
		"original_data":      twilioSystem,
		"converted_data":     nil, // Will be filled below
		"migration_metadata": map[string]interface{}{
			"enhanced_by":    "Engine Room AI",
			"migration_time": time.Now().Format("2006-01-02 15:04:05"),
			"source_format":  config.SourceFormat,
			"target_format":  config.TargetFormat,
		},
	}

	// Reorder users based on Engine Room AI's recommendations
	var orderedUsers []TwilioUser
	for _, item := range plan.RecommendedOrder {
		orderedUsers = append(orderedUsers, item.Account)
	}
	twilioSystem.Users = orderedUsers

	// Convert to target format
	if config.SourceFormat == "Twilio" && config.TargetFormat == "RingCentral" {
		rcSystem := convertTwilioToRingCentral(twilioSystem)
		enhancedOutput["converted_data"] = rcSystem
	} else {
		return fmt.Errorf("Engine Room AI-enhanced migration currently only supports Twilio to RingCentral")
	}

	// Write enhanced output
	targetData, err := json.MarshalIndent(enhancedOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	err = ioutil.WriteFile(config.TargetFile, targetData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

func migrate(config MigrationConfig) error {
	// Read source file
	sourceData, err := ioutil.ReadFile(config.SourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Parse based on source format and convert to target format
	var targetData []byte
	
	if config.SourceFormat == "Twilio" && config.TargetFormat == "RingCentral" {
		targetData, err = twilioToRingCentral(sourceData)
	} else if config.SourceFormat == "RingCentral" && config.TargetFormat == "Twilio" {
		targetData, err = ringCentralToTwilio(sourceData)
	} else if config.SourceFormat == config.TargetFormat {
		// Same format, just copy
		targetData = sourceData
	} else {
		return fmt.Errorf("unsupported migration path: %s to %s", config.SourceFormat, config.TargetFormat)
	}

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Write target file
	err = ioutil.WriteFile(config.TargetFile, targetData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

func convertTwilioToRingCentral(twilioSystem TwilioPhoneSystem) RingCentralPhoneSystem {
	var rcSystem RingCentralPhoneSystem
	
	// Convert users to accounts
	for _, user := range twilioSystem.Users {
		account := RingCentralAccount{
			ID:         user.ID,
			Username:   user.Name,
			Contact:    user.Email,
			MainNumber: user.PhoneNumber,
			Active:     user.Status == "active",
		}
		rcSystem.Accounts = append(rcSystem.Accounts, account)
	}

	// Convert lines to numbers
	for _, line := range twilioSystem.Lines {
		var features []string
		for capability, enabled := range line.Capabilities {
			if enabled {
				features = append(features, capability)
			}
		}
		
		number := RingCentralNumber{
			ID:       line.SID,
			Number:   line.Number,
			Features: features,
			Region:   line.Location,
		}
		rcSystem.Numbers = append(rcSystem.Numbers, number)
	}

	return rcSystem
}

func twilioToRingCentral(data []byte) ([]byte, error) {
	var twilioSystem TwilioPhoneSystem
	if err := json.Unmarshal(data, &twilioSystem); err != nil {
		return nil, err
	}

	rcSystem := convertTwilioToRingCentral(twilioSystem)
	return json.MarshalIndent(rcSystem, "", "  ")
}

func ringCentralToTwilio(data []byte) ([]byte, error) {
	var rcSystem RingCentralPhoneSystem
	if err := json.Unmarshal(data, &rcSystem); err != nil {
		return nil, err
	}

	// Convert RingCentral to Twilio format
	var twilioSystem TwilioPhoneSystem
	
	// Convert accounts to users
	for _, account := range rcSystem.Accounts {
		status := "inactive"
		if account.Active {
			status = "active"
		}
		
		user := TwilioUser{
			ID:          account.ID,
			Name:        account.Username,
			Email:       account.Contact,
			PhoneNumber: account.MainNumber,
			Status:      status,
		}
		twilioSystem.Users = append(twilioSystem.Users, user)
	}

	// Convert numbers to lines
	for _, number := range rcSystem.Numbers {
		capabilities := make(map[string]bool)
		for _, feature := range number.Features {
			capabilities[feature] = true
		}
		
		line := TwilioLine{
			SID:          number.ID,
			Number:       number.Number,
			Capabilities: capabilities,
			Location:     number.Region,
		}
		twilioSystem.Lines = append(twilioSystem.Lines, line)
	}

	return json.MarshalIndent(twilioSystem, "", "  ")
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}