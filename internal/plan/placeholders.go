package plan

import "strings"

const (
	PlaceholderPendingStepExecution      = "PENDING_STEP_EXECUTION"
	PlaceholderPendingStepReview         = "PENDING_STEP_REVIEW"
	PlaceholderPendingUntilArchive       = "PENDING_UNTIL_ARCHIVE"
	PlaceholderUpdateRequiredAfterReopen = "UPDATE_REQUIRED_AFTER_REOPEN"
)

func containsArchivePlaceholderToken(content string) bool {
	return strings.Contains(content, PlaceholderPendingUntilArchive) ||
		strings.Contains(content, PlaceholderUpdateRequiredAfterReopen)
}
