package server

const ScrapeStatusGood = "good"
const ScrapeStatusPending = "pending"
const ScrapeStatusBad = "bad"

func getValidStatuses() []string {
	return []string{
		ScrapeStatusGood,
		ScrapeStatusPending,
		ScrapeStatusBad,
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
