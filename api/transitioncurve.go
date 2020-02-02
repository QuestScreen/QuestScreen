package api

import (
	"math"
	"time"
)

// TransitionCurve provides various curves useful for smooth transitions.
// All its methods return a float between 0.0 and 1.0 calculated from the
// given elapsed time in relation to the total duration.
type TransitionCurve struct {
	// Duration sets the length of the transition
	Duration time.Duration
}

// Linear implements a linear transition curve.
func (tc TransitionCurve) Linear(elapsed time.Duration) float32 {
	return float32(elapsed) / float32(tc.Duration)
}

// Cubic implements a cubic transition curve.
func (tc TransitionCurve) Cubic(elapsed time.Duration) float32 {
	x := float64(elapsed) / float64(tc.Duration)
	return float32(-2.0*math.Pow(x, 3) + 3.0*math.Pow(x, 2))
}
