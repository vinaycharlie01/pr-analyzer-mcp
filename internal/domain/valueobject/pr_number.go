package valueobject

import "fmt"

type PRNumber struct {
	value int
}

func NewPRNumber(n int) (PRNumber, error) {
	if n <= 0 {
		return PRNumber{}, fmt.Errorf("PR number must be positive, got %d", n)
	}
	return PRNumber{value: n}, nil
}

func (p PRNumber) Value() int    { return p.value }
func (p PRNumber) String() string { return fmt.Sprintf("#%d", p.value) }
