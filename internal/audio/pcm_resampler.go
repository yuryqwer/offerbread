package audio

import (
	"encoding/binary"
	"math"
)

const (
	// FIR filter: Blackman-windowed sinc low-pass
	// 48kHz → 16kHz = 3:1 decimation
	// Cutoff at 7.5kHz (below 8kHz Nyquist for 16kHz output)
	firTaps      = 48
	firCutoffHz  = 7500.0
	firInputRate = 48000.0
	decimFactor  = 3
)

// PCMResampler converts 48kHz stereo int16 LE PCM to 16kHz mono int16 LE PCM.
//
// Processing pipeline per stereo frame:
//  1. Deinterleave L/R int16 → mono float64 (average)
//  2. Feed into FIR low-pass anti-aliasing filter (ring buffer)
//  3. 3:1 decimation — output every 3rd filtered sample
//  4. Clamp and encode back to int16 LE
//
// Maintains internal FIR state across Process() calls for seamless streaming.
type PCMResampler struct {
	coeffs  [firTaps]float64 // precomputed FIR coefficients
	history [firTaps]float64 // ring buffer of last N mono samples
	histIdx int              // write cursor in history ring
	phase   int              // decimation phase: 0..decimFactor-1
}

// NewPCMResampler creates a resampler with precomputed Blackman-windowed sinc FIR coefficients.
func NewPCMResampler() *PCMResampler {
	r := &PCMResampler{}
	r.initCoeffs()
	return r
}

// initCoeffs computes Blackman-windowed sinc low-pass FIR coefficients.
// Normalized for unity gain at DC.
func (r *PCMResampler) initCoeffs() {
	fc := firCutoffHz / firInputRate // 7500/48000 = 0.15625
	n := float64(firTaps)
	mid := (n - 1) / 2.0

	for i := range firTaps {
		x := float64(i) - mid

		// Blackman window
		w := 0.42 -
			0.5*math.Cos(2*math.Pi*float64(i)/(n-1)) +
			0.08*math.Cos(4*math.Pi*float64(i)/(n-1))

		// Sinc function (L'Hôpital's rule at x=0)
		var sinc float64
		if x == 0 {
			sinc = 2 * fc
		} else {
			sinc = math.Sin(2*math.Pi*fc*x) / (math.Pi * x)
		}

		r.coeffs[i] = w * sinc
	}

	// Normalize to unity gain at DC
	var sum float64
	for i := range firTaps {
		sum += r.coeffs[i]
	}
	for i := range firTaps {
		r.coeffs[i] /= sum
	}
}

// Process converts a chunk of 48kHz stereo int16 LE PCM bytes to 16kHz mono int16 LE PCM bytes.
//
// Input: raw bytes from screencast audio capture (48kHz, stereo, s16le).
// Output: resampled 16kHz mono s16le bytes. May be empty if the input chunk
// is too small to produce a full decimated output sample.
//
// Trailing bytes that don't form a complete stereo frame (<4 bytes) are discarded.
func (r *PCMResampler) Process(input []byte) []byte {
	frameCount := len(input) / 4
	if frameCount == 0 {
		return nil
	}

	// Pre-allocate output buffer: ~frameCount/3 + margin
	outSamples := make([]int16, 0, frameCount/decimFactor+1)

	for i := range frameCount {
		off := i * 4
		left := int16(binary.LittleEndian.Uint16(input[off : off+2]))
		right := int16(binary.LittleEndian.Uint16(input[off+2 : off+4]))

		// Stereo → mono: average left and right channels
		mono := (float64(left) + float64(right)) / 2.0

		// Push into FIR history ring buffer
		r.history[r.histIdx] = mono
		r.histIdx = (r.histIdx + 1) % firTaps

		// Decimation: only compute FIR output every 3rd input sample
		r.phase++
		if r.phase == decimFactor {
			r.phase = 0

			var filtered float64
			for j := range firTaps {
				// Index into history ring in reverse chronological order:
				// coeffs[0] * newest, coeffs[1] * prev, ..., coeffs[N-1] * oldest
				idx := (r.histIdx - 1 - j) % firTaps
				if idx < 0 {
					idx += firTaps
				}
				filtered += r.coeffs[j] * r.history[idx]
			}

			outSamples = append(outSamples, float64ToInt16(filtered))
		}
	}

	// Encode int16 samples back to little-endian bytes
	output := make([]byte, len(outSamples)*2)
	for i, s := range outSamples {
		binary.LittleEndian.PutUint16(output[i*2:], uint16(s))
	}

	return output
}

// Reset clears the internal FIR history and decimation phase.
// Call when restarting a capture session to avoid stale filter state.
func (r *PCMResampler) Reset() {
	for i := range r.history {
		r.history[i] = 0
	}
	r.histIdx = 0
	r.phase = 0
}

// float64ToInt16 clamps a float64 sample to the int16 range and converts.
func float64ToInt16(f float64) int16 {
	if f > 32767 {
		return 32767
	}
	if f < -32768 {
		return -32768
	}
	return int16(f)
}
