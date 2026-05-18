package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/robert-nix/ansihtml"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/cockroachdb/pebble/v2"
)

const defaultNodeTTL = 15 * time.Second

//go:embed templates/dashboard.html
var templateFS embed.FS

var pageTmpl = template.Must(
	template.ParseFS(templateFS, "templates/dashboard.html"),
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI256)
}

// ── Stats collection ──────────────────────────────────────────────────────────

func collectStats(hostname, webURL, version string, ttl time.Duration) (NodeStats, error) {
	cpuPcts, err := cpu.Percent(200*time.Millisecond, true)
	if err != nil {
		return NodeStats{}, err
	}
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return NodeStats{}, err
	}
	loadStat, err := load.Avg()
	if err != nil {
		return NodeStats{}, err
	}
	return NodeStats{
		Name:       hostname,
		Version:    version,
		WebURL:     webURL,
		TTLSeconds: int(ttl / time.Second),
		CPU:        cpuPcts,
		MemUsed:    vmStat.Used,
		MemTotal:   vmStat.Total,
		Load:       [3]float64{loadStat.Load1, loadStat.Load5, loadStat.Load15},
		UpdatedAt:  time.Now().UnixNano(),
	}, nil
}

// ── ANSI rendering ────────────────────────────────────────────────────────────

const barWidth = 20

var (
	styleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleBlue   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type barSegment struct {
	Until float64
	Style lipgloss.Style
}

var (
	cpuSegments = []barSegment{
		{Until: 55, Style: styleGreen},
		{Until: 80, Style: styleYellow},
		{Until: 100, Style: styleRed},
	}
	memSegments = []barSegment{
		{Until: 35, Style: styleGreen},
		{Until: 70, Style: styleBlue},
		{Until: 90, Style: styleYellow},
		{Until: 100, Style: styleRed},
	}
)

type healthState string

const (
	healthFresh   healthState = "fresh"
	healthStale   healthState = "stale"
	healthOffline healthState = "offline"
)

type nodeHealthInfo struct {
	State healthState
	Age   time.Duration
	TTL   time.Duration
}

func nodeTTL(s NodeStats) time.Duration {
	if s.TTLSeconds > 0 {
		return time.Duration(s.TTLSeconds) * time.Second
	}
	return defaultNodeTTL
}

func nodeHealth(s NodeStats) nodeHealthInfo {
	ttl := nodeTTL(s)
	if s.UpdatedAt == 0 {
		return nodeHealthInfo{State: healthOffline, TTL: ttl}
	}
	age := time.Since(time.Unix(0, s.UpdatedAt))
	if age > ttl {
		return nodeHealthInfo{State: healthOffline, Age: age, TTL: ttl}
	}
	if age > ttl/2 {
		return nodeHealthInfo{State: healthStale, Age: age, TTL: ttl}
	}
	return nodeHealthInfo{State: healthFresh, Age: age, TTL: ttl}
}

func nodeOnline(s NodeStats) bool {
	return nodeHealth(s).State != healthOffline
}

func pctBar(pct float64, width int, segments []barSegment) string {
	filled := int(math.Round(pct / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	var sb strings.Builder
	for pos := 0; pos < filled; pos++ {
		style := segmentStyle((float64(pos)+1)/float64(width)*100, segments)
		sb.WriteString(style.Render("█"))
	}
	sb.WriteString(styleDim.Render(strings.Repeat("░", width-filled)))
	return sb.String()
}

func segmentStyle(pct float64, segments []barSegment) lipgloss.Style {
	for _, segment := range segments {
		if pct <= segment.Until {
			return segment.Style
		}
	}
	return styleRed
}

func renderANSI(s NodeStats) string {
	var sb strings.Builder
	health := nodeHealth(s)

	status := styleGreen.Render("●")
	if health.State == healthStale {
		status = styleYellow.Render("●")
	}
	if health.State == healthOffline {
		status = styleRed.Render("●")
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", status, s.Name))
	if health.State == healthOffline {
		sb.WriteString(styleDim.Render("  offline"))
		sb.WriteByte('\n')
		return sb.String()
	}
	if health.State == healthStale {
		sb.WriteString(fmt.Sprintf("  stale %.0fs ago\n", health.Age.Seconds()))
	} else {
		sb.WriteString(fmt.Sprintf("  updated %.0fs ago\n", health.Age.Seconds()))
	}
	sb.WriteString(styleDim.Render(strings.Repeat("─", barWidth+14)))
	sb.WriteByte('\n')

	for i, pct := range s.CPU {
		bar := pctBar(pct, barWidth, cpuSegments)
		sb.WriteString(fmt.Sprintf("CPU%-2d [%s] %5.1f%%\n", i, bar, pct))
	}

	memPct := 0.0
	if s.MemTotal > 0 {
		memPct = float64(s.MemUsed) / float64(s.MemTotal) * 100
	}
	sb.WriteString(fmt.Sprintf("Mem   [%s] %5.1f%%\n", pctBar(memPct, barWidth, memSegments), memPct))
	sb.WriteString(fmt.Sprintf("      %s / %s\n", fmtBytes(s.MemUsed), fmtBytes(s.MemTotal)))

	loadStyle := styleGreen
	if s.Load[0] > 2.0 {
		loadStyle = styleYellow
	}
	if s.Load[0] > 4.0 {
		loadStyle = styleRed
	}
	sb.WriteString(fmt.Sprintf("Load  %s  %.2f  %s  %.2f\n",
		loadStyle.Render("▶"), s.Load[0], styleCyan.Render(fmt.Sprintf("%.2f", s.Load[1])), s.Load[2]))

	return sb.String()
}

func fmtBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ── Layout ────────────────────────────────────────────────────────────────────

type layoutParams struct {
	CellWidth float64
	FontSize  float64
}

func computeLayout(nodeCount, winW, winH int) layoutParams {
	if nodeCount == 0 {
		nodeCount = 1
	}
	aspect := 16.0 / 9.0
	if winW > 0 && winH > 0 {
		aspect = float64(winW) / float64(winH)
	}
	cols := int(math.Round(math.Sqrt(float64(nodeCount) * aspect)))
	if cols < 1 {
		cols = 1
	}
	if cols > nodeCount {
		cols = nodeCount
	}
	cw := 100.0 / float64(cols)
	return layoutParams{CellWidth: cw, FontSize: cw * 0.016}
}
func avgCPU(s NodeStats) float64 {
	if len(s.CPU) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s.CPU {
		sum += v
	}
	return sum / float64(len(s.CPU))
}

func maxCPU(s NodeStats) float64 {
	max := 0.0
	for _, v := range s.CPU {
		if v > max {
			max = v
		}
	}
	return max
}

func computeRefreshIntervalMs(nodes []NodeStats) int {
	if len(nodes) == 0 {
		return 3000
	}
	maxCPU := 0.0
	maxLoad := 0.0
	for _, s := range nodes {
		if !nodeOnline(s) {
			continue
		}
		if cpu := avgCPU(s); cpu > maxCPU {
			maxCPU = cpu
		}
		if s.Load[0] > maxLoad {
			maxLoad = s.Load[0]
		}
	}

	switch {
	case maxCPU < 30 && maxLoad < 1.0:
		return 2000
	case maxCPU < 55 && maxLoad < 2.0:
		return 3500
	case maxCPU < 80 && maxLoad < 4.0:
		return 6500
	default:
		return 11000
	}
}

func findBestNodeHint(nodes []NodeStats) string {
	best := (*NodeStats)(nil)
	bestScore := math.MaxFloat64
	for i := range nodes {
		s := &nodes[i]
		if !nodeOnline(*s) {
			continue
		}
		score := nodeScore(*s)
		if score < bestScore {
			bestScore = score
			best = s
		}
	}
	if best == nil {
		return "no responsive peers yet"
	}
	return fmt.Sprintf("lowest-load node: %s (%.0f%% cpu, %.2f load)", best.Name, avgCPU(*best), best.Load[0])
}

type clusterSummary struct {
	Fresh   int
	Stale   int
	Offline int
	Hottest string
	HotCPU  float64
	HotLoad float64
	HasHot  bool
}

func summarizeCluster(nodes []NodeStats) clusterSummary {
	var summary clusterSummary
	hotScore := -1.0
	for _, s := range nodes {
		switch nodeHealth(s).State {
		case healthFresh:
			summary.Fresh++
		case healthStale:
			summary.Stale++
			continue
		default:
			summary.Offline++
			continue
		}
		cpu := avgCPU(s)
		score := cpu + s.Load[0]*10
		if score > hotScore {
			hotScore = score
			summary.Hottest = s.Name
			summary.HotCPU = cpu
			summary.HotLoad = s.Load[0]
			summary.HasHot = true
		}
	}
	return summary
}

func (s clusterSummary) TerminalHeader() string {
	hot := "hottest: none"
	if s.HasHot {
		hot = fmt.Sprintf("hottest: %s %.0f%% cpu %.2f load", s.Hottest, s.HotCPU, s.HotLoad)
	}
	return fmt.Sprintf("online %d - stale %d - offline %d - %s", s.Fresh, s.Stale, s.Offline, hot)
}

func nodeScore(s NodeStats) float64 {
	return avgCPU(s) + s.Load[0]*10
}

func findNode(nodes []NodeStats, name string) *NodeStats {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
	}
	return nil
}

func findLowerLoadRedirect(nodes []NodeStats, selfName string) *NodeStats {
	self := findNode(nodes, selfName)
	if self == nil || !nodeOnline(*self) {
		return nil
	}
	selfScore := nodeScore(*self)
	var best *NodeStats
	bestScore := math.MaxFloat64
	for i := range nodes {
		s := &nodes[i]
		if s.Name == selfName || s.WebURL == "" || !nodeOnline(*s) {
			continue
		}
		score := nodeScore(*s)
		if score < bestScore {
			bestScore = score
			best = s
		}
	}
	if best == nil {
		return nil
	}
	if bestScore <= selfScore*0.70 && selfScore-bestScore >= 10 {
		return best
	}
	return nil
}

func pageURL(base string, values url.Values) string {
	base = normalizePageBase(base)
	if base == "" {
		base = "/"
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "/"
	}
	query := parsed.Query()
	for key, vals := range values {
		query.Del(key)
		for _, value := range vals {
			query.Add(key, value)
		}
	}
	encoded := query.Encode()
	parsed.RawQuery = encoded
	if encoded == "" {
		return parsed.String()
	}
	return parsed.String()
}

func normalizePageBase(base string) string {
	if base == "" || strings.HasPrefix(base, "/") {
		return base
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "/"
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

func displayQuery(r *http.Request, winW, winH int) url.Values {
	values := url.Values{}
	for _, key := range []string{"theme", "palette"} {
		if value := r.URL.Query().Get(key); value != "" {
			values.Set(key, value)
		}
	}
	values.Set("w", fmt.Sprintf("%d", winW))
	values.Set("h", fmt.Sprintf("%d", winH))
	return values
}

// ── HTTP handler ──────────────────────────────────────────────────────────────

type cellData struct {
	Name     string
	URL      string
	HTML     template.HTML
	State    healthState
	CPUAvg   float64
	CPUMax   float64
	MemPct   float64
	MemUsed  uint64
	MemTotal uint64
	Load1    float64
	Load5    float64
	Load15   float64
	Age      float64
	Link     bool
}

type pageData struct {
	Layout       layoutParams
	Nodes        []cellData
	RefreshMs    int
	RefreshLabel string
	RefreshURL   string
	BestHint     string
	Summary      clusterSummary
}

func makeHandler(db *pebble.DB, selfName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		winW, winH := 0, 0
		fmt.Sscanf(r.URL.Query().Get("w"), "%d", &winW)
		fmt.Sscanf(r.URL.Query().Get("h"), "%d", &winH)

		nodes, err := dbScanAll(db)
		if err != nil {
			http.Error(w, "db error", 500)
			return
		}

		layout := computeLayout(len(nodes), winW, winH)
		cells := make([]cellData, 0, len(nodes))
		for _, s := range nodes {
			htmlBytes := ansihtml.ConvertToHTML([]byte(renderANSI(s)))
			health := nodeHealth(s)
			nodeURL := ""
			if s.WebURL != "" {
				nodeURL = pageURL(s.WebURL, displayQuery(r, winW, winH))
			}
			memPct := 0.0
			if s.MemTotal > 0 {
				memPct = float64(s.MemUsed) / float64(s.MemTotal) * 100
			}

			cells = append(cells, cellData{
				Name:     s.Name,
				URL:      nodeURL,
				HTML:     template.HTML(htmlBytes),
				State:    health.State,
				CPUAvg:   avgCPU(s),
				CPUMax:   maxCPU(s),
				MemPct:   memPct,
				MemUsed:  s.MemUsed,
				MemTotal: s.MemTotal,
				Load1:    s.Load[0],
				Load5:    s.Load[1],
				Load15:   s.Load[2],
				Age:      health.Age.Seconds(),
				Link:     nodeURL != "",
			})
		}

		refreshMs := computeRefreshIntervalMs(nodes)
		bestHint := findBestNodeHint(nodes)
		refreshValues := displayQuery(r, winW, winH)
		refreshURL := pageURL("/", refreshValues)
		if redirectNode := findLowerLoadRedirect(nodes, selfName); redirectNode != nil {
			refreshURL = pageURL(redirectNode.WebURL, refreshValues)
		}

		var buf bytes.Buffer
		if err := pageTmpl.Execute(&buf, pageData{
			Layout:       layout,
			Nodes:        cells,
			RefreshMs:    refreshMs,
			RefreshLabel: fmt.Sprintf("%.1fs", float64(refreshMs)/1000),
			RefreshURL:   refreshURL,
			BestHint:     bestHint,
			Summary:      summarizeCluster(nodes),
		}); err != nil {
			http.Error(w, "template error", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	}
}
