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
	out := make([]Duplicate, 0)
	for i := 0; i < len(occurrences); i++ {
		for j := i + 1; j < len(occurrences); j++ {
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
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Similarity > out[j].Similarity })
	return out
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
