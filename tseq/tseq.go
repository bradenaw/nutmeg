// package tseq provides a facility for testing all control-flow permutations.
//
// See https://qumulo.com/blog/making-100-code-coverage-as-easy-as-flipping-a-coin/ for background.
package tseq

import (
	"encoding/hex"
	"flag"
	"regexp"
	"testing"
	"time"
)

var flagScript string

func init() {
	flag.StringVar(&flagScript, "tseq.run", "", "Rerun just one failed scenario from a tseq test.")
}

func AllPermutations(t *testing.T, f func(tseq *TSeq)) {
	tseq := &TSeq{}

	if flagScript != "" {
		script, err := scriptFromString(flagScript)
		if err != nil {
			t.Fatalf("--tseq.run provided but doesn't parse: %s", err)
		}
		t.Logf("--tseq.run provided, running just the scenario: %s", flagScript)
		tseq.script = script
		runOne(t, tseq, f)
		return
	}

	lastReport := time.Now()
	i := 0
	for {
		runOne(t, tseq, f)
		if !tseq.next() {
			break
		}

		i++
		if time.Since(lastReport) > time.Second {
			t.Logf("ran %d permutations", i)
			lastReport = time.Now()
		}
	}
	t.Logf("ran %d permutations", i)
}

func runOne(t *testing.T, tseq *TSeq, f func(tseq *TSeq)) {
	defer func() {
		if t.Failed() {
			t.Logf(
				"rerun just this scenario with --test.run=%q --tseq.run=%s",
				"^"+regexp.QuoteMeta(t.Name())+"$",
				scriptToString(tseq.script),
			)
			t.Log(
				"note that changing the control flow of your program's calls to tseq will make " +
					"this flag not test the same scenario",
			)
		}
	}()

	f(tseq)

	if tseq.i < len(tseq.script) {
		t.Fatalf("it looks like this test is not deterministic")
	}
}

type TSeq struct {
	script []bool
	i      int
}

func (tseq *TSeq) Either() bool {
	if tseq.i == len(tseq.script) {
		tseq.script = append(tseq.script, false)
	}
	next := tseq.script[tseq.i]
	tseq.i++
	return next
}

func (tseq *TSeq) Choose(n int) int {
	min := 0
	max := n - 1

	for max > min {
		mid := min + (max-min)/2
		if tseq.Either() {
			min = mid + 1
		} else {
			max = mid
		}
	}
	return min
}

func (tseq *TSeq) next() bool {
	tseq.script = trimTrailing(tseq.script, true)
	if len(tseq.script) > 0 {
		tseq.script[len(tseq.script)-1] = true
		tseq.i = 0
		return true
	}
	return false
}

func trimTrailing(script []bool, b bool) []bool {
	for len(script) > 0 && script[len(script)-1] == b {
		script = script[:len(script)-1]
	}
	return script
}

func scriptToString(script []bool) string {
	// Safe to trim trailing falses because that's what tseq will choose anyway.
	script = trimTrailing(script, false)
	b := make([]byte, (len(script)+7)/8)
	for i := range script {
		if script[i] {
			b[i/8] |= 1 << (i % 8)
		}
	}
	return hex.EncodeToString(b)
}

func scriptFromString(s string) ([]bool, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	script := make([]bool, len(b)*8)
	for i := range script {
		script[i] = ((b[i/8] >> (i % 8)) & 1) == 1
	}
	// Safe to trim trailing falses because that's what tseq will choose anyway.
	script = trimTrailing(script, false)
	return script, nil
}
