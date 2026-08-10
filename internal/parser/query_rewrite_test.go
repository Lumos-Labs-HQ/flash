package parser

import "testing"

// TestRewriteINListToANY_Renumbering guards the $N renumbering done by
// rewriteINListToANY. Several of these cases FAIL on buggy renumbering that
// collapses distinct params onto the same number.
func TestRewriteINListToANY_Renumbering(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"trailing params after in-list",
			"SELECT * FROM t WHERE id IN ($1, $2) AND status = $3 AND kind = $4",
			"SELECT * FROM t WHERE id = ANY($1) AND status = $2 AND kind = $3",
		},
		{
			"in-list in the middle",
			"SELECT * FROM t WHERE status = $1 AND id IN ($2, $3) AND kind = $4",
			"SELECT * FROM t WHERE status = $1 AND id = ANY($2) AND kind = $3",
		},
		{
			"two in-lists plus trailing param",
			"SELECT * FROM t WHERE a IN ($1, $2) AND b IN ($3, $4) AND c = $5",
			"SELECT * FROM t WHERE a = ANY($1) AND b = ANY($2) AND c = $3",
		},
		{
			"three-element in-list then trailing",
			"SELECT * FROM t WHERE id IN ($1, $2, $3) AND status = $4 AND kind = $5",
			"SELECT * FROM t WHERE id = ANY($1) AND status = $2 AND kind = $3",
		},
		{
			"single-element in-list is left untouched",
			"SELECT * FROM t WHERE id IN ($1) AND x = $2",
			"SELECT * FROM t WHERE id IN ($1) AND x = $2",
		},
		{
			"no in-list is a no-op",
			"SELECT * FROM t WHERE a = $1 AND b = $2",
			"SELECT * FROM t WHERE a = $1 AND b = $2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rewriteINListToANY(tt.in); got != tt.want {
				t.Errorf("rewriteINListToANY mismatch:\n  in   %q\n  got  %q\n  want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestRewriteINListToANY_ParamsStaySequential asserts that after the rewrite
// the surviving $N params are distinct and gap-free (1..N). A renumber bug
// that drops or duplicates a param number is caught here.
func TestRewriteINListToANY_ParamsStaySequential(t *testing.T) {
	got := rewriteINListToANY("SELECT * FROM t WHERE id IN ($1, $2, $3) AND status = $4 AND kind = $5")
	nums := extractOrderedParamNums(got)
	want := []int{1, 2, 3}
	if len(nums) != len(want) {
		t.Fatalf("param set = %v, want %v (sql=%q)", nums, want, got)
	}
	for i := range want {
		if nums[i] != want[i] {
			t.Fatalf("param set = %v, want %v (sql=%q)", nums, want, got)
		}
	}
}
