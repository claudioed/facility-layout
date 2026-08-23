package shared

// Capacity is the physical envelope of a storage location: the maximum
// weight (kg) and maximum volume (m3) it may hold. It is a value object;
// both dimensions must be strictly positive, because a slot that can hold
// nothing is not a slot.
type Capacity struct {
	maxWeightKg float64
	maxVolumeM3 float64
}

// NewCapacity validates and constructs a Capacity envelope.
func NewCapacity(maxWeightKg, maxVolumeM3 float64) (Capacity, error) {
	if maxWeightKg <= 0 {
		return Capacity{}, ErrInvalidMaxWeight
	}
	if maxVolumeM3 <= 0 {
		return Capacity{}, ErrInvalidMaxVolume
	}
	return Capacity{maxWeightKg: maxWeightKg, maxVolumeM3: maxVolumeM3}, nil
}

// MaxWeightKg returns the maximum weight this envelope permits, in kilograms.
func (c Capacity) MaxWeightKg() float64 { return c.maxWeightKg }

// MaxVolumeM3 returns the maximum volume this envelope permits, in cubic metres.
func (c Capacity) MaxVolumeM3() float64 { return c.maxVolumeM3 }

// IsZero reports whether this is the zero Capacity, i.e. "no override
// supplied — fall back to the LocationType's default envelope".
func (c Capacity) IsZero() bool {
	return c == Capacity{}
}
