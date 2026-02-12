package evaluation

// RecallAtK computes Recall@K: the fraction of relevant items found in the top-K retrieved results.
// Returns 0.0 if relevant is empty.
func RecallAtK(relevant, retrieved []string, k int) float64 {
	if len(relevant) == 0 {
		return 0.0
	}

	relevantSet := make(map[string]struct{}, len(relevant))
	for _, r := range relevant {
		relevantSet[r] = struct{}{}
	}

	topK := retrieved
	if k < len(topK) {
		topK = topK[:k]
	}

	found := 0
	for _, r := range topK {
		if _, ok := relevantSet[r]; ok {
			found++
		}
	}

	return float64(found) / float64(len(relevant))
}

// MRRAtK computes Mean Reciprocal Rank at K: the reciprocal of the rank of the first relevant item
// in the top-K retrieved results. Returns 0.0 if no relevant item is found in top-K.
func MRRAtK(relevant, retrieved []string, k int) float64 {
	if len(relevant) == 0 || len(retrieved) == 0 {
		return 0.0
	}

	relevantSet := make(map[string]struct{}, len(relevant))
	for _, r := range relevant {
		relevantSet[r] = struct{}{}
	}

	topK := retrieved
	if k < len(topK) {
		topK = topK[:k]
	}

	for i, r := range topK {
		if _, ok := relevantSet[r]; ok {
			return 1.0 / float64(i+1)
		}
	}

	return 0.0
}
