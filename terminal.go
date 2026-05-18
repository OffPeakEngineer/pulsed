package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/pebble/v2"
)

const (
	terminalCellWidth = 42
	terminalCellGap   = 2
)

var terminalCellStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("8")).
	Padding(0, 1).
	Width(terminalCellWidth)

func terminalRenderLoop(db *pebble.DB, listMode bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		renderTerminalSnapshot(db, listMode)
		<-ticker.C
	}
}

func renderTerminalSnapshot(db *pebble.DB, listMode bool) {
	nodes, err := dbScanAll(db)
	if err != nil {
		log.Printf("terminal render: %v", err)
		return
	}

	fmt.Print("\033[H\033[2J")
	if len(nodes) == 0 {
		fmt.Println("psstd terminal mirror")
		fmt.Println(styleDim.Render("waiting for cluster state..."))
		return
	}

	fmt.Printf("psstd terminal mirror - %d node(s) - %s\n", len(nodes), time.Now().Format(time.RFC3339))
	fmt.Println(summarizeCluster(nodes).TerminalHeader())
	fmt.Println()
	if listMode {
		fmt.Print(renderTerminalNodes(nodes))
		return
	}
	fmt.Print(renderTerminalGrid(nodes, terminalWidth()))
}

func renderTerminalNodes(nodes []NodeStats) string {
	sortNodesByName(nodes)

	var sb strings.Builder
	for i, node := range nodes {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(renderANSI(node))
	}
	return sb.String()
}

func renderTerminalGrid(nodes []NodeStats, width int) string {
	sortNodesByName(nodes)
	if len(nodes) == 0 {
		return ""
	}

	cols := terminalColumns(width)
	rows := make([]string, 0, (len(nodes)+cols-1)/cols)
	for start := 0; start < len(nodes); start += cols {
		end := start + cols
		if end > len(nodes) {
			end = len(nodes)
		}
		cells := make([]string, 0, end-start)
		for _, node := range nodes[start:end] {
			cells = append(cells, renderTerminalCell(node))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return strings.Join(rows, "\n") + "\n"
}

func renderTerminalCell(node NodeStats) string {
	return terminalCellStyle.Render(strings.TrimRight(renderANSI(node), "\n"))
}

func terminalColumns(width int) int {
	cellAndGap := terminalCellWidth + terminalCellGap
	if width < cellAndGap {
		return 1
	}
	cols := (width + terminalCellGap) / cellAndGap
	if cols < 1 {
		return 1
	}
	return cols
}

func terminalWidth() int {
	if value := os.Getenv("COLUMNS"); value != "" {
		if width, err := strconv.Atoi(value); err == nil && width > 0 {
			return width
		}
	}
	return 100
}

func sortNodesByName(nodes []NodeStats) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
}
