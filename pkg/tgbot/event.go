package tgbot

type Event interface {
	Type() EvType
	Desc() any
}

type EvType int

const (
	EvTypeHailProbabilityChange = EvType(iota)
)

type eventHailProbabilityChange struct {
	oldLevel, newLevel int
	region             string
}

func (e eventHailProbabilityChange) Type() EvType {
	return EvTypeHailProbabilityChange
}

func (e eventHailProbabilityChange) Desc() any {
	return e
}

func NewEventHailProbabilityChange(oldLevel, newLevel int, region string) Event {
	return eventHailProbabilityChange{
		oldLevel: oldLevel,
		newLevel: newLevel,
		region:   region,
	}
}
