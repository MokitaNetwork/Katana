package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrependKatanaIfUnique(t *testing.T) {
	require := require.New(t)
	tcs := []struct {
		in  []string
		out []string
	}{
		// Should prepend "ukatana" to a slice of denoms, unless it is already present.
		{[]string{}, []string{"ukatana"}},
		{[]string{"a"}, []string{"ukatana", "a"}},
		{[]string{"x", "a", "heeeyyy"}, []string{"ukatana", "x", "a", "heeeyyy"}},
		{[]string{"x", "a", "ukatana"}, []string{"x", "a", "ukatana"}},
	}
	for i, tc := range tcs {
		require.Equal(tc.out, prependKatanaIfUnique(tc.in), i)
	}

}
