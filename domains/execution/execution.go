package execution

import (
	"time"

	"github.com/TTRSQ/bbwrapper/domains/base"
	"github.com/TTRSQ/bbwrapper/domains/order/id"
)

// Execution information of someone's excution.
type Execution struct {
	id.ID
	base.Norm
	IsBuy     bool
	OccuredAt time.Time
}
