package effects

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

var csvHeader = []string{
	"gu_id", "rank", "gu_type", "primary_role", "entry_status", "mvp",
	"block_id", "trigger", "target", "operations", "block_status", "reasons", "proposal",
}

// WriteCSV emits one row per block (or one row for an entry without blocks).
func WriteCSV(w io.Writer, report Report) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(csvHeader); err != nil {
		return err
	}
	for _, entry := range report.Entries {
		if len(entry.Blocks) == 0 {
			if err := writer.Write([]string{
				entry.ID, strconv.Itoa(entry.Rank), entry.Type, entry.PrimaryRole, string(entry.Status),
				strconv.FormatBool(entry.MVP), "", "", "", "", "", strings.Join(entry.Reasons, ";"), "",
			}); err != nil {
				return err
			}
			continue
		}
		for _, block := range entry.Blocks {
			operations := make([]string, 0, len(block.Operations))
			for _, operation := range block.Operations {
				operations = append(operations, operation.Op)
			}
			if err := writer.Write([]string{
				entry.ID, strconv.Itoa(entry.Rank), entry.Type, entry.PrimaryRole, string(entry.Status),
				strconv.FormatBool(entry.MVP), block.ID, block.Trigger, block.Target,
				strings.Join(operations, "|"), string(block.Status), strings.Join(block.Reasons, ";"), block.Proposal,
			}); err != nil {
				return err
			}
		}
	}
	writer.Flush()
	return writer.Error()
}
