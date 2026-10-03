package demo

import "testing"

func TestOutcomes(t *testing.T) {
	counts := map[string]int{}
	for seq := uint64(1); seq <= 20; seq++ {
		counts[Outcome(seq)]++
	}
	if counts["ack"] != 16 || counts["nak-once"] != 3 || counts["term"] != 1 {
		t.Fatalf("unexpected 20-job cycle: %v", counts)
	}
}
