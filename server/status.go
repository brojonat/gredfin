package server

const ScrapeStatusGood = "good"
const ScrapeStatusPending = "pending"

func getValidStatuses() []string {
	return []string{
		ScrapeStatusGood,
		ScrapeStatusPending,
	}
}

func isValidStatus(v string) bool {
	for _, s := range getValidStatuses() {
		if v == s {
			return true
		}
	}
	return false
}
