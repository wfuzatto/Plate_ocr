package normalize

import (
	"sort"
	"strings"
	"unicode"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

const Version = "br-v1"

var alphaToDigit = map[rune]rune{'O':'0','Q':'0','D':'0','I':'1','L':'1','T':'1','Z':'2','S':'5','G':'6','B':'8'}
var digitToAlpha = map[rune]rune{'0':'O','1':'I','2':'Z','5':'S','6':'G','8':'B'}

type pattern struct{ name, mask string }
var patterns = []pattern{{"BR_MERCOSUL","LLLDLDD"},{"BR_OLD","LLLDDDD"}}

func Sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' { b.WriteRune(r) }
	}
	return b.String()
}

func Candidates(raw string, baseConfidence float64, limit int) []domain.Candidate {
	clean := Sanitize(raw)
	if clean == "" { return nil }
	if baseConfidence < 0 { baseConfidence = 0 }
	if baseConfidence > 1 { baseConfidence = 1 }
	if limit < 1 { limit = 8 }

	out := make([]domain.Candidate, 0, 4)
	if len(clean) == 7 {
		for _, p := range patterns {
			if c, ok := applyPattern(raw, clean, p, baseConfidence); ok { out = append(out, c) }
		}
	}
	if len(out) == 0 {
		out = append(out, domain.Candidate{RawText:raw, NormalizedText:clean, Format:"UNKNOWN", Confidence:baseConfidence*.45})
	}
	uniq := make(map[string]domain.Candidate)
	for _, c := range out {
		k := c.NormalizedText + "|" + c.Format
		if old, ok := uniq[k]; !ok || c.Confidence > old.Confidence { uniq[k] = c }
	}
	out = out[:0]
	for _, c := range uniq { out = append(out, c) }
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence == out[j].Confidence { return out[i].Format < out[j].Format }
		return out[i].Confidence > out[j].Confidence
	})
	if len(out) > limit { out = out[:limit] }
	return out
}

func applyPattern(raw, clean string, p pattern, base float64) (domain.Candidate, bool) {
	runes := []rune(clean)
	mask := []rune(p.mask)
	corrections := []string{}
	penalty := 0.0
	for i, r := range runes {
		want := mask[i]
		if want == 'L' {
			if unicode.IsLetter(r) && r < 128 { continue }
			if repl, ok := digitToAlpha[r]; ok {
				corrections = append(corrections, string(r)+"->"+string(repl))
				runes[i] = repl
				penalty += .08
				continue
			}
			return domain.Candidate{}, false
		}
		if want == 'D' {
			if r >= '0' && r <= '9' { continue }
			if repl, ok := alphaToDigit[r]; ok {
				corrections = append(corrections, string(r)+"->"+string(repl))
				runes[i] = repl
				penalty += .08
				continue
			}
			return domain.Candidate{}, false
		}
	}
	conf := base - penalty
	if conf < 0 { conf = 0 }
	return domain.Candidate{
		RawText:raw, NormalizedText:string(runes), Format:p.name,
		Confidence:conf, Corrections:corrections,
	}, true
}
