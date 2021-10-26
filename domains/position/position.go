package position

import (
	"github.com/TTRSQ/bbwrapper/domains/base"
)

type Position struct {
	Symbol string
	Long   []base.Norm
	Short  []base.Norm
	// UpdatedAt time.Time
}

func (p *Position) HasLong() bool {
	return len(p.Long) != 0
}

func (p *Position) HasShort() bool {
	return len(p.Short) != 0
}
