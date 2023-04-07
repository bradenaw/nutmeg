package tseq

import (
	"fmt"
	"testing"
)

func TestAllPermutations(t *testing.T) {
	type boolAndInt struct {
		b bool
		i int
	}
	var actual []boolAndInt
	AllPermutations(t, func(tseq *TSeq) {
		b := tseq.Either()
		i := tseq.Choose(5)
		actual = append(actual, boolAndInt{b, i})
	})

	expected := []boolAndInt{
		{false, 0},
		{false, 1},
		{false, 2},
		{false, 3},
		{false, 4},
		{true, 0},
		{true, 1},
		{true, 2},
		{true, 3},
		{true, 4},
	}

	n := len(expected)
	if len(actual) > len(expected) {
		n = len(actual)
	}

	str := func(b []boolAndInt, i int) string {
		if i >= len(b) {
			return "<missing>"
		}
		return fmt.Sprintf("%v", b[i])
	}
	t.Logf("actual              expected")
	for i := 0; i < n; i++ {
		t.Logf("%-20s%s", str(actual, i), str(expected, i))
		if i >= len(actual) || i >= len(expected) || actual[i] != expected[i] {
			t.Fail()
		}
	}
}

func TestRoundtripScript(t *testing.T) {
	check := func(script []bool) {
		encoded := scriptToString(script)
		decoded, err := scriptFromString(encoded)
		if err != nil {
			t.Fatal(err)
		}

		if fmt.Sprintf("%v", script) != fmt.Sprintf("%v", decoded) {
			t.Logf("script  %v", script)
			t.Logf("encoded %s", encoded)
			t.Logf("decoded %v", decoded)
			t.Fatal("did not roundtrip")
		}
	}

	check([]bool{})
	check([]bool{true})
	check([]bool{false})
	check([]bool{false, true})
	check([]bool{false, true, false})
	check([]bool{
		true, true, false, false, true, true, false, false,
		true, false, true, false, true, false, true, false,
		false, true, false, true, false, true, false, true,
		true, false, true,
	})
}

// To test the flags manually.
func TestFail(t *testing.T) {
	AllPermutations(t, func(tseq *TSeq) {
		if tseq.Choose(40000) == 31231 {
			t.FailNow()
		}
	})
}
