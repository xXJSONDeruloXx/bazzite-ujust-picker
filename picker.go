package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Version information - will be set at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type recipe struct {
	name        string
	description string
}

type model struct {
	categories     []string
	recipesByCat   map[string][]recipe
	currentTab     int
	selectedRecipe int
}

var runRecipe string

var (
	layoutWidth    = 72
	appBorder      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Width(layoutWidth)
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	catSelected    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00afff")).Underline(true)
	catInactive    = lipgloss.NewStyle().Faint(true)
	recipeActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	recipeInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	descText       = lipgloss.NewStyle().Padding(1, 0).Width(layoutWidth - 4)
	controlStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

func divider() string {
	return strings.Repeat("─", layoutWidth-4)
}

func initialModel() model {
	recipeDir := "/usr/share/ublue-os/just"
	files, _ := filepath.Glob(filepath.Join(recipeDir, "*.just"))

	categories := []string{}
	recipesByCat := make(map[string][]recipe)

	recipeRegex := regexp.MustCompile(`^\s*([a-zA-Z0-9_-]+)\s*:\s*.*`)
	commentRegex := regexp.MustCompile(`^\s*#\s*(.*)`)

	for _, file := range files {
		base := filepath.Base(file)
		if strings.Contains(base, "picker") {
			continue
		}
		category := cleanCategoryName(base)

		f, _ := os.Open(file)
		defer f.Close()

		scanner := bufio.NewScanner(f)
		var lastComment string
		found := false
		for scanner.Scan() {
			line := scanner.Text()

			if commentRegex.MatchString(line) {
				lastComment = commentRegex.FindStringSubmatch(line)[1]
				continue
			}

			if matches := recipeRegex.FindStringSubmatch(line); matches != nil {
				name := matches[1]
				if strings.HasPrefix(name, "_") || strings.Contains(line, "alias") || strings.Contains(line, "[private]") {
					continue
				}
				recipesByCat[category] = append(recipesByCat[category], recipe{
					name:        name,
					description: lastComment,
				})
				lastComment = ""
				found = true
			}
		}
		if found {
			categories = append(categories, category)
		}
	}

	sort.Strings(categories)
	return model{
		categories:     categories,
		recipesByCat:   recipesByCat,
		currentTab:     0,
		selectedRecipe: 0,
	}
}

func cleanCategoryName(filename string) string {
	name := strings.TrimSuffix(filename, ".just")
	name = regexp.MustCompile(`^[0-9]+-`).ReplaceAllString(name, "")
	parts := strings.Split(name, "-")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, " ")
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			if m.currentTab > 0 {
				m.currentTab--
				m.selectedRecipe = 0
			}
		case "right":
			if m.currentTab < len(m.categories)-1 {
				m.currentTab++
				m.selectedRecipe = 0
			}
		case "up":
			if m.selectedRecipe > 0 {
				m.selectedRecipe--
			}
		case "down":
			if m.selectedRecipe < len(m.currentRecipes())-1 {
				m.selectedRecipe++
			}
		case "enter":
			selected := m.currentRecipes()[m.selectedRecipe]
			runRecipe = selected.name
			return m, tea.Quit
		case "esc", "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	var header strings.Builder
	header.WriteString(titleStyle.Render("Available ujust recipes") + "\n")
	header.WriteString(divider() + "\n")

	left := ""
	if m.currentTab > 0 {
		left = "← " + m.categories[m.currentTab-1]
	}
	center := catSelected.Render(m.categories[m.currentTab])
	right := ""
	if m.currentTab < len(m.categories)-1 {
		right = m.categories[m.currentTab+1] + " →"
	}

	navLine := fmt.Sprintf("%-20s %-30s %20s", left, center, right)
	header.WriteString(navLine + "\n")
	header.WriteString(controlStyle.Render("← → Change Category | ↑ ↓ Navigate Recipes | Enter: Select | Esc: Exit") + "\n")
	header.WriteString(divider() + "\n")

	var recipeLines []string
	for i, r := range m.currentRecipes() {
		line := r.name
		if i == m.selectedRecipe {
			recipeLines = append(recipeLines, recipeActive.Render("▶ "+line))
		} else {
			recipeLines = append(recipeLines, recipeInactive.Render("  "+line))
		}
	}

	var descBlock string
	if len(m.currentRecipes()) > 0 {
		r := m.currentRecipes()[m.selectedRecipe]
		if r.description != "" {
			desc := "Selected: " + recipeActive.Render(r.name) + "\n\n" + wrap(r.description, layoutWidth-4)
			descBlock = divider() + "\n" + descText.Render(desc)
		}
	}

	fullView := header.String() + "\n" + strings.Join(recipeLines, "\n") + "\n\n" + descBlock
	return appBorder.Render(fullView)
}

func (m model) currentRecipes() []recipe {
	return m.recipesByCat[m.categories[m.currentTab]]
}

func wrap(s string, limit int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	line := ""

	for _, word := range words {
		if len(line)+len(word)+1 > limit {
			result.WriteString(line + "\n")
			line = word
		} else {
			if line != "" {
				line += " "
			}
			line += word
		}
	}

	result.WriteString(line)
	return result.String()
}

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("ujust-picker version %s, commit %s, built at %s\n", version, commit, date)
		return
	}

	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if runRecipe != "" {
		fmt.Print("\033[2J\033[H")
		fmt.Printf("Running recipe: %s...\n\n", runRecipe)
		cmd := exec.Command("ujust", runRecipe)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}
