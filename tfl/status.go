package tfl

import (
	"cmp"
	"errors"
	"fmt"
	"ptt/output"
	"slices"
	"strings"

	"github.com/fatih/color"
)

// The TfL API assigns numerical codes to each status severity, and it seems like you could *mostly* get by just
// treating lower numbers as being more severe, but that may not necessarily work in all cases. So below is a list of
// most severity descriptions observed at https://api.tfl.gov.uk/Line/Meta/Severity (excluding some that seem clearly
// only intended for use with stations, rather than lines), ordered roughly in the order of severity.
var severityOrder = []string{
	"Closed",
	"No Service",
	"Not Running",
	"Planned Closure",
	"Suspended",
	"Part Closure",
	"Part Closed",
	"Part Suspended",
	"Severe Delays",
	// Special Service is used differently on different lines and can mean anything from minor delays to suspended.
	"Special Service",
	"Reduced Service",
	"Bus Service",
	"Change of frequency",
	"Diverted",
	"Issues Reported",
	"Minor Delays",
	"Information",
	"No Issues",
	"Good Service",
}

func lineStatusUrl(lines []string) (string, error) {
	if len(lines) == 0 {
		return "", errors.New("no lines provided")
	}
	return fmt.Sprintf("%s/Line/%s/Status", BaseUrl, strings.Join(lines, ",")), nil
}

type LineStatus struct {
	Description string  `json:"statusSeverityDescription"`
	Reason      *string `json:"reason,omitempty"`

	// `severity` is the internal value that we assign to the status, based on the position of the description in the
	// `severityOrder` slice. It is different to the numerical value assigned by the TfL API. `severityInit` describes
	// whether we have already calculated and cached the result.
	severity     int
	severityInit bool
}

func (status *LineStatus) Severity() int {
	if !status.severityInit {
		status.severity = slices.Index(severityOrder, status.Description)
		status.severityInit = true
	}
	return status.severity
}

func (status *LineStatus) severityColor() *color.Color {
	// https://api.tfl.gov.uk/Line/Meta/Severity
	var key string
	s := status.Severity()
	if s <= 8 {
		key = "red"
	} else if s <= 16 {
		key = "yellow"
	} else {
		key = "green"
	}
	rgb, ok := safetyColors[key]
	if ok {
		return rgb.Add(color.Bold)
	} else {
		return nil
	}
}

func (line *Line) mostSevereStatus() (*LineStatus, error) {
	if len(line.Statuses) == 0 {
		return nil, errors.New("no statuses found")
	}
	mostSevere := slices.MinFunc(line.Statuses, func(a, b *LineStatus) int {
		return cmp.Compare(a.Severity(), b.Severity())
	})
	return mostSevere, nil
}

func (line *Line) lineColor() *color.Color {
	var lineColor *color.Color
	var ok bool
	lineColor, ok = lineColors[line.Id]
	if !ok {
		lineColor, ok = modeColors[line.Mode]
	}
	if ok {
		return lineColor
	} else {
		return nil
	}
}

func (line *Line) ToRowWithStatus(withColor bool) (output.Row, error) {
	lineCell := output.Cell{}
	statusCell := output.Cell{}
	row := output.Row{}
	mostSevere, err := line.mostSevereStatus()
	if err != nil {
		return row, err
	}
	lineColor := line.lineColor()
	severityColor := mostSevere.severityColor()
	if lineColor != nil && withColor {
		lineCell.AddText("    ", lineColor)
		lineCell.AddText(" ", nil)
	}
	lineCell.AddText(line.Name, nil)
	statusCell.AddText(mostSevere.Description, severityColor)
	row.AddCell(lineCell)
	row.AddCell(statusCell)
	return row, nil
}

func GetLineStatuses(lineIds []string, apiKey string) ([]Line, error) {
	url, err := lineStatusUrl(lineIds)
	if err != nil {
		return nil, err
	}
	lines, err := request[[]Line](url, apiKey)
	if err != nil {
		return nil, err
	}
	return lines, nil
}
