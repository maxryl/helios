package ui

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

//go:embed pev2.html
var pev2HTML string

const explainPrefix = "EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT JSON) "

// RunExplainAnalyze executes EXPLAIN ANALYZE on the current query and opens
// the result in the system browser using pev2 for visualization.
func RunExplainAnalyze(t *Terminal) {
	sql := t.editor.SelectedText()
	if sql == "" {
		sql = t.editor.Text
	}
	if strings.TrimSpace(sql) == "" {
		return
	}

	t.statusLabel.SetText("Explaining...")

	go func() {
		ctx := context.Background()
		querier := t.querier()
		explainSQL := explainPrefix + sql

		rows, err := querier.Query(ctx, explainSQL)
		if err != nil {
			fyne.Do(func() {
				t.statusLabel.SetText(fmt.Sprintf("Explain error: %s", err))
				dialog.ShowError(fmt.Errorf("EXPLAIN ANALYZE failed: %w", err), t.window)
			})
			return
		}
		defer rows.Close()

		var planJSON json.RawMessage
		if rows.Next() {
			if err := rows.Scan(&planJSON); err != nil {
				fyne.Do(func() {
					t.statusLabel.SetText(fmt.Sprintf("Explain error: %s", err))
					dialog.ShowError(fmt.Errorf("EXPLAIN ANALYZE scan failed: %w", err), t.window)
				})
				return
			}
		}

		// Build temp HTML: pev2 + auto-load script.
		planStr := string(planJSON)
		escapedPlan := html.EscapeString(planStr)
		escapedQuery := html.EscapeString(sql)

		inject := fmt.Sprintf(`<script>
document.addEventListener('DOMContentLoaded', function() {
	setTimeout(function() {
		if (window.setPlanData) {
			window.setPlanData(
				'Helios EXPLAIN ANALYZE',
				%s,
				%s
			);
		}
	}, 100);
});
</script>`, mustJSONString(planStr), mustJSONString(sql))

		_ = escapedPlan
		_ = escapedQuery

		htmlContent := strings.Replace(pev2HTML, "</body>", inject+"</body>", 1)

		tmpDir := os.TempDir()
		tmpFile := filepath.Join(tmpDir, "helios_explain.html")
		if err := os.WriteFile(tmpFile, []byte(htmlContent), 0644); err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("failed to write temp file: %w", err), t.window)
			})
			return
		}

		openBrowser(tmpFile)

		fyne.Do(func() {
			t.statusLabel.SetText("Explain opened in browser")
		})
	}()
}

func mustJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
