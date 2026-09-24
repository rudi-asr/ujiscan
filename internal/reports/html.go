// Package reports provides HTML and PDF rendering
package reports

import (
	"fmt"
	"strings"
)

// HTMLRenderer converts reports to HTML
type HTMLRenderer struct {
	brandLogo   string
	brandColor  string
	clientName  string
}

// NewHTMLRenderer creates a renderer
func NewHTMLRenderer(logo string, color string) *HTMLRenderer {
	return &HTMLRenderer{
		brandLogo:  logo,
		brandColor: color,
	}
}

// Render converts a report to HTML
func (r *HTMLRenderer) Render(report *Report) string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Assessment Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            background: #f5f5f5;
        }
        
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: white;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        
        .header p {
            font-size: 1.1em;
            opacity: 0.9;
        }
        
        .content {
            padding: 40px;
        }
        
        .section {
            margin-bottom: 40px;
            border-bottom: 2px solid #eee;
            padding-bottom: 30px;
        }
        
        .section:last-child {
            border-bottom: none;
        }
        
        .section h2 {
            font-size: 1.8em;
            margin-bottom: 20px;
            color: #667eea;
        }
        
        .summary-box {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin-bottom: 20px;
        }
        
        .info-box {
            padding: 15px;
            background: #f9f9f9;
            border-left: 4px solid #667eea;
        }
        
        .info-box strong {
            display: block;
            color: #667eea;
            margin-bottom: 5px;
        }
        
        .severity-grid {
            display: grid;
            grid-template-columns: repeat(5, 1fr);
            gap: 15px;
            margin: 20px 0;
        }
        
        .severity-box {
            text-align: center;
            padding: 20px;
            border-radius: 8px;
            font-weight: bold;
        }
        
        .severity-critical {
            background: #ffebee;
            color: #c62828;
            border: 2px solid #c62828;
        }
        
        .severity-high {
            background: #fff3e0;
            color: #e65100;
            border: 2px solid #e65100;
        }
        
        .severity-medium {
            background: #fffde7;
            color: #f57f17;
            border: 2px solid #f57f17;
        }
        
        .severity-low {
            background: #e8f5e9;
            color: #2e7d32;
            border: 2px solid #2e7d32;
        }
        
        .severity-info {
            background: #e3f2fd;
            color: #1565c0;
            border: 2px solid #1565c0;
        }
        
        .severity-box .count {
            font-size: 2em;
            display: block;
            margin-bottom: 5px;
        }
        
        .finding {
            margin: 20px 0;
            padding: 20px;
            border: 1px solid #ddd;
            border-radius: 4px;
            border-left: 4px solid #667eea;
        }
        
        .finding.critical {
            border-left-color: #c62828;
            background: #ffebee;
        }
        
        .finding.high {
            border-left-color: #e65100;
            background: #fff3e0;
        }
        
        .finding.medium {
            border-left-color: #f57f17;
            background: #fffde7;
        }
        
        .finding.low {
            border-left-color: #2e7d32;
            background: #e8f5e9;
        }
        
        .finding.info {
            border-left-color: #1565c0;
            background: #e3f2fd;
        }
        
        .finding-title {
            font-size: 1.3em;
            font-weight: bold;
            margin-bottom: 10px;
        }
        
        .finding-meta {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 15px;
            margin-bottom: 15px;
            font-size: 0.9em;
        }
        
        .finding-meta strong {
            color: #667eea;
        }
        
        .remediation {
            background: #f0f4ff;
            padding: 15px;
            border-radius: 4px;
            margin-top: 15px;
        }
        
        .remediation h4 {
            color: #667eea;
            margin-bottom: 10px;
        }
        
        .remediation ul {
            margin-left: 20px;
        }
        
        .remediation li {
            margin-bottom: 8px;
        }
        
        .footer {
            background: #f5f5f5;
            padding: 20px;
            text-align: center;
            font-size: 0.9em;
            color: #999;
            border-top: 1px solid #ddd;
        }
        
        .metrics {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 15px;
            margin: 20px 0;
        }
        
        .metric-item {
            padding: 15px;
            background: #f9f9f9;
            border-radius: 4px;
        }
        
        .metric-value {
            font-size: 1.5em;
            font-weight: bold;
            color: #667eea;
        }
        
        .metric-label {
            font-size: 0.9em;
            color: #999;
            margin-top: 5px;
        }
        
        .risk-gauge {
            font-size: 3em;
            font-weight: bold;
            text-align: center;
            margin: 20px 0;
            padding: 20px;
            border-radius: 4px;
            background: #f9f9f9;
        }
        
        .page-break {
            page-break-after: always;
            margin-bottom: 40px;
        }
        
        @media print {
            body {
                background: white;
            }
            .container {
                box-shadow: none;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>☠️ Security Assessment Report</h1>
            <p>Agentic Penetration Testing Results</p>
        </div>
        
        <div class="content">
` + r.renderExecutiveSummary(report) + `
            
` + r.renderFindingsSummary(report) + `
            
` + r.renderFindings(report) + `
            
` + r.renderPlaybookExecution(report) + `
            
` + r.renderRemediationPlan(report) + `
        </div>
        
        <div class="footer">
            <p>Report generated by ujiscan v0.1.0 | ` + report.Metadata.GeneratedAt.Format("2006-01-02 15:04:05") + `</p>
            <p>Report ID: ` + report.Metadata.ReportID + `</p>
        </div>
    </div>
</body>
</html>`

	return html
}

// renderExecutiveSummary renders engagement summary
func (r *HTMLRenderer) renderExecutiveSummary(report *Report) string {
	html := `
        <div class="section">
            <h2>Executive Summary</h2>
            <div class="summary-box">
                <div class="info-box">
                    <strong>Target</strong>
                    ` + report.Engagement.Target + `
                </div>
                <div class="info-box">
                    <strong>Assessment Date</strong>
                    ` + report.Engagement.CompletedAt.Format("2006-01-02") + `
                </div>
                <div class="info-box">
                    <strong>Scope</strong>
                    ` + strings.Join(report.Engagement.Scope, ", ") + `
                </div>
                <div class="info-box">
                    <strong>Duration</strong>
                    ` + fmt.Sprintf("%d minutes", report.Engagement.DurationMins) + `
                </div>
            </div>
            
            <p style="margin: 20px 0; font-size: 1.1em;">
                Assessment identified <strong>` + fmt.Sprintf("%d", report.Summary.FindingsCount) + ` vulnerabilities</strong> 
                including <strong>` + fmt.Sprintf("%d", report.Summary.SeverityDistribution.Critical) + ` critical issues</strong> 
                requiring immediate remediation.
            </p>
            
            <div class="risk-gauge" style="color: ` + getRiskColor(report.Summary.RiskScore) + `;">
                Risk Level: ` + strings.ToUpper(report.Summary.RiskLevel) + ` (` + fmt.Sprintf("%.1f", report.Summary.RiskScore) + `/10)
            </div>
        </div>
`
	return html
}

// renderFindingsSummary renders severity distribution
func (r *HTMLRenderer) renderFindingsSummary(report *Report) string {
	sev := report.Summary.SeverityDistribution
	
	html := `
        <div class="section">
            <h2>Findings Summary</h2>
            
            <div class="severity-grid">
                <div class="severity-box severity-critical">
                    <span class="count">` + fmt.Sprintf("%d", sev.Critical) + `</span>
                    <span>🔴 Critical</span>
                </div>
                <div class="severity-box severity-high">
                    <span class="count">` + fmt.Sprintf("%d", sev.High) + `</span>
                    <span>🟠 High</span>
                </div>
                <div class="severity-box severity-medium">
                    <span class="count">` + fmt.Sprintf("%d", sev.Medium) + `</span>
                    <span>🟡 Medium</span>
                </div>
                <div class="severity-box severity-low">
                    <span class="count">` + fmt.Sprintf("%d", sev.Low) + `</span>
                    <span>🟢 Low</span>
                </div>
                <div class="severity-box severity-info">
                    <span class="count">` + fmt.Sprintf("%d", sev.Info) + `</span>
                    <span>ℹ️ Info</span>
                </div>
            </div>
            
            <div class="metrics">
                <div class="metric-item">
                    <div class="metric-value">` + fmt.Sprintf("%.1f", report.Summary.Metrics.MeanCVSS) + `</div>
                    <div class="metric-label">Mean CVSS Score</div>
                </div>
                <div class="metric-item">
                    <div class="metric-value">` + fmt.Sprintf("%d", report.Summary.Metrics.ExploitableCount) + `</div>
                    <div class="metric-label">Exploitable Findings</div>
                </div>
            </div>
        </div>
`
	return html
}

// renderFindings renders detailed findings
func (r *HTMLRenderer) renderFindings(report *Report) string {
	html := `
        <div class="section">
            <h2>Detailed Findings</h2>
`

	for i, f := range report.Findings {
		html += fmt.Sprintf(`
            <div class="finding %s">
                <div class="finding-title">
                    %d. %s
                </div>
                
                <div class="finding-meta">
                    <div><strong>Severity:</strong> %s</div>
                    <div><strong>CWE:</strong> %s</div>
                    <div><strong>CVSS:</strong> %s</div>
                </div>
                
                <p><strong>Description:</strong> %s</p>
                <p><strong>Impact:</strong> %s</p>
                
                <div class="remediation">
                    <h4>Remediation Steps:</h4>
                    <ul>
`, i+1, f.Title, f.Severity, f.CWE, f.CVSSV3, f.Description, f.Impact)

		for _, step := range f.Remediation.Steps {
			html += fmt.Sprintf(`                        <li>%s</li>
`, step)
		}

		html += fmt.Sprintf(`                    </ul>
                    <p style="margin-top: 10px;"><strong>Estimated Effort:</strong> %s</p>
                </div>
            </div>
`, f.Remediation.EstimatedEffort)
	}

	html += `
        </div>
`
	return html
}

// renderPlaybookExecution renders execution details
func (r *HTMLRenderer) renderPlaybookExecution(report *Report) string {
	html := `
        <div class="section">
            <h2>Assessment Methodology</h2>
            <p>Executed using <strong>` + report.Playbook.PlaybookName + `</strong> playbook:</p>
            
            <div style="margin: 20px 0;">
`

	for _, phase := range report.Playbook.Phases {
		html += fmt.Sprintf(`
                <div style="margin: 15px 0; padding: 15px; background: #f9f9f9; border-radius: 4px;">
                    <strong>%s</strong> (%ds)<br>
                    Tools: %s<br>
                    Findings: %d
                </div>
`, phase.Name, phase.DurationSecs, strings.Join(phase.Tools, ", "), phase.FindingsCount)
	}

	html += `
            </div>
        </div>
`
	return html
}

// renderRemediationPlan renders timeline
func (r *HTMLRenderer) renderRemediationPlan(report *Report) string {
	html := `
        <div class="section">
            <h2>Remediation Roadmap</h2>
            
            <div style="margin: 20px 0;">
`

	for _, item := range report.Remediation.Timeline {
		html += fmt.Sprintf(`
                <div style="margin: 15px 0; padding: 15px; background: #f9f9f9; border-radius: 4px;">
                    <strong>Week %d: %s</strong><br>
                    <ul style="margin-left: 20px; margin-top: 10px;">
`, item.Week, item.Milestone)

		for _, task := range item.Tasks {
			html += fmt.Sprintf(`                        <li>%s</li>
`, task)
		}

		html += `                    </ul>
                </div>
`
	}

	html += `
            </div>
        </div>
`
	return html
}

// getRiskColor returns color based on risk score
func getRiskColor(score float32) string {
	switch {
	case score >= 9:
		return "#c62828" // red
	case score >= 7:
		return "#e65100" // orange
	case score >= 4:
		return "#f57f17" // yellow
	case score >= 2:
		return "#2e7d32" // green
	default:
		return "#1565c0" // blue
	}
}
