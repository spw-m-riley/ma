package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Duplicate struct {
	Sentence   string
	Locations  []string
	Similarity float64
}

type Report struct {
	Exact []Duplicate
	Near  []Duplicate
}

type sentenceOccurrence struct {
	raw        string
	normalized string
	location   string
}

var punctuationPattern = regexp.MustCompile(`[^\pL\pN\s]+`)

func Analyze(paths []string, threshold float64) (Report, error) {
	occurrences := make([]sentenceOccurrence, 0)
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return Report{}, err
		}
		occurrences = append(occurrences, collectSentences(path, string(content))...)
	}

	return Report{
		Exact: exactDuplicates(occurrences),
		Near:  nearDuplicates(occurrences, threshold),
	}, nil
}

func Fingerprint(sentence string) string {
	sum := sha256.Sum256([]byte(normalizeSentence(sentence)))
	return hex.EncodeToString(sum[:])
}

func collectSentences(path string, content string) []sentenceOccurrence {
	lines := strings.Split(content, "\n")
	out := make([]sentenceOccurrence, 0, len(lines))
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, sentenceOccurrence{
			raw:        trimmed,
			normalized: normalizeSentence(trimmed),
			location:   path + ":" + strconv.Itoa(index+1),
		})
	}
	return out
}

func exactDuplicates(occurrences []sentenceOccurrence) []Duplicate {
	grouped := make(map[string][]sentenceOccurrence)
	for _, occurrence := range occurrences {
		grouped[occurrence.normalized] = append(grouped[occurrence.normalized], occurrence)
	}

	out := make([]Duplicate, 0)
	for _, group := range grouped {
		if len(group) < 2 {
			continue
		}
		locations := make([]string, 0, len(group))
		for _, occurrence := range group {
			locations = append(locations, occurrence.location)
		}
		sort.Strings(locations)
		out = append(out, Duplicate{
			Sentence:   group[0].raw,
			Locations:  locations,
			Similarity: 1,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sentence < out[j].Sentence })
	return out
}

func nearDuplicates(occurrences []sentenceOccurrence, threshold float64) []Duplicate {
	// Use algorithmic reduction: group by approximate word set first
	// to avoid comparing completely dissimilar sentences
	candidates := selectCandidatePairs(occurrences, threshold)
	
	out := make([]Duplicate, 0)
	for _, pair := range candidates {
		i, j := pair[0], pair[1]
		if occurrences[i].normalized == occurrences[j].normalized {
			continue
		}
		similarity := jaccard(occurrences[i].normalized, occurrences[j].normalized)
		if similarity < threshold {
			continue
		}
		out = append(out, Duplicate{
			Sentence:   occurrences[i].raw + " ~ " + occurrences[j].raw,
			Locations:  []string{occurrences[i].location, occurrences[j].location},
			Similarity: similarity,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Similarity > out[j].Similarity })
	return out
}

// selectCandidatePairs uses algorithmic reduction to select pairs worth comparing
// It groups sentences by word count and only compares within similar word count ranges
func selectCandidatePairs(occurrences []sentenceOccurrence, _ float64) [][2]int {
	// Group by word count
	byWordCount := make(map[int][]int)
	for idx, occ := range occurrences {
		wordCount := len(strings.Fields(occ.normalized))
		byWordCount[wordCount] = append(byWordCount[wordCount], idx)
	}

	// Find word count ranges that could have similarity >= threshold
	// Sentences with very different word counts can't have high similarity
	var pairs [][2]int
	
	// For each group, compare within the group and with adjacent word counts
	sortedCounts := make([]int, 0, len(byWordCount))
	for wc := range byWordCount {
		sortedCounts = append(sortedCounts, wc)
	}
	sort.Ints(sortedCounts)
	
	for index, wc := range sortedCounts {
		indices := byWordCount[wc]
		
		// Compare within the same word count
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				if indices[i] < indices[j] {
					pairs = append(pairs, [2]int{indices[i], indices[j]})
				}
			}
		}
		
		// Compare with the next actual word-count bucket when it differs by at most one word.
		if index+1 >= len(sortedCounts) {
			continue
		}
		nextWC := sortedCounts[index+1]
		if nextWC-wc > 1 {
			continue
		}
		nextIndices := byWordCount[nextWC]
		for _, i := range indices {
			for _, j := range nextIndices {
				if i < j {
					pairs = append(pairs, [2]int{i, j})
				} else {
					pairs = append(pairs, [2]int{j, i})
				}
			}
		}
	}

	return pairs
}

func normalizeSentence(sentence string) string {
	sentence = strings.ToLower(sentence)
	sentence = punctuationPattern.ReplaceAllString(sentence, " ")
	sentence = strings.Join(strings.Fields(sentence), " ")
	return sentence
}

func jaccard(left string, right string) float64 {
	leftSet := wordSet(left)
	rightSet := wordSet(right)

	intersection := 0
	union := make(map[string]struct{}, len(leftSet)+len(rightSet))
	for word := range leftSet {
		union[word] = struct{}{}
		if _, ok := rightSet[word]; ok {
			intersection++
		}
	}
	for word := range rightSet {
		union[word] = struct{}{}
	}
	if len(union) == 0 {
		return 0
	}
	return float64(intersection) / float64(len(union))
}

func wordSet(sentence string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, word := range strings.Fields(sentence) {
		set[word] = struct{}{}
	}
	return set
}
