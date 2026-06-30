package reporter

import "fmt"

func BuildReportID(nodeID int64, periodStart int64, kind string) string {
	return fmt.Sprintf("%d-%d-%s", nodeID, periodStart, kind)
}
