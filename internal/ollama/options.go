package ollama

import (
	"os"
	"strconv"
)

// Decode profiles and generation budgets.
//
// Two things go wrong when we let a model's own Modelfile PARAMETER lines
// decide these values.
//
// Budgets: Ollama applies the model's num_ctx / num_predict when the request
// omits them. Keyvan/german-ocr-turbo ships num_ctx 4096 and num_predict 2048,
// so a multi-page document silently overflowed the window and a dense page was
// truncated mid-transcription — invisibly, because done_reason was discarded.
// Every call now states its own budget.
//
// Sampling: turbo also ships temperature 0.1 / top_k 20 / top_p 0.9, which is
// sampling rather than greedy, so the same page can transcribe differently
// twice. For transcription and translation we want the most likely token and a
// reproducible result, which is also what the author chose for the newer
// german-ocr-3 models ("greedy für reproduzierbaren Output").
//
// repeat_penalty deserves its own note: turbo ships 1.5. The penalty only
// applies over a sliding window (repeat_last_n, 64 by default), so it does not
// prevent a model from repeating a whole page hundreds of tokens later, while
// it does push the model away from legitimately repeated wording. Real letters
// repeat themselves. We lower it and detect long-range repetition separately
// (see repetition.go).

const (
	defaultVisionNumCtx      = 8192
	defaultVisionNumPredict  = 4096
	defaultTextNumCtxFloor   = 8192
	defaultTextNumCtxCeiling = 32768
	defaultTextNumPredictMin = 1024
	defaultRepeatPenalty     = 1.1

	// visionEndpointChat is the default because models published with
	// RENDERER/PARSER rather than TEMPLATE (turbo declares
	// RENDERER qwen3-vl-instruct) are templated by Ollama's renderer, which
	// drives /api/chat. On /api/generate the two paths are not equivalent
	// (ollama#14793), and an untemplated instruct model continues the
	// document instead of answering — the repetition loop we observed.
	visionEndpointChat     = "/api/chat"
	visionEndpointGenerate = "/api/generate"

	// avgCharsPerToken is a deliberately conservative estimate for German
	// text. German compounds tokenize badly (Veranlagungszeitraum,
	// Identifikationsnummer), so underestimating here means overflowing the
	// context window, which is the failure we are removing.
	avgCharsPerToken = 3
)

// baseOptions returns the generation knobs shared by every call: greedy
// decode, a tamed repeat penalty, and an optional thread count for the
// num_thread sweep. Callers add num_ctx and num_predict themselves, since
// those depend on what is being sent.
func baseOptions() map[string]any {
	opts := map[string]any{
		"temperature":    0,
		"top_k":          1,
		"repeat_penalty": envFloat("OLLAMA_REPEAT_PENALTY", defaultRepeatPenalty),
	}
	// Ollama defaults its thread count to performance cores only
	// (ollama#6264), which is why a 14-core hybrid CPU sits near 22%. That
	// default is often correct — generation is memory-bandwidth bound and
	// E-cores add contention — so this is a knob to sweep, not to raise
	// blindly. Unset means "let Ollama decide".
	if n := envInt("OLLAMA_NUM_THREAD", 0); n > 0 {
		opts["num_thread"] = n
	}
	return opts
}

// visionOptions returns the options for a per-page transcription call.
func visionOptions() map[string]any {
	opts := baseOptions()
	opts["num_ctx"] = envIntMin("OLLAMA_NUM_CTX", defaultVisionNumCtx, 2048)
	opts["num_predict"] = envInt("OLLAMA_NUM_PREDICT_VISION", defaultVisionNumPredict)
	return opts
}

// textOptions returns the options for a text call, sized from the input.
func textOptions(inputChars int) map[string]any {
	opts := baseOptions()
	opts["num_ctx"] = textNumCtx(inputChars)
	opts["num_predict"] = textNumPredict(inputChars)
	return opts
}

// textNumCtx sizes the context window to hold the input, an output of
// comparable length (a translation is roughly as long as its source), and
// prompt overhead.
func textNumCtx(inputChars int) int {
	est := (inputChars/avgCharsPerToken)*2 + 1024
	floor := envInt("OLLAMA_NUM_CTX_TEXT_FLOOR", defaultTextNumCtxFloor)
	ceiling := envInt("OLLAMA_NUM_CTX_TEXT", defaultTextNumCtxCeiling)
	return clamp(est, floor, ceiling)
}

// textNumPredict bounds the output. A translation of N input tokens lands
// near N, so we allow generous headroom rather than a tight cap: the point
// is to stop a runaway generation, not to trim legitimate output. When the
// budget is genuinely hit, done_reason reports "length" and the caller
// retries with more room instead of accepting truncated text.
func textNumPredict(inputChars int) int {
	est := (inputChars/avgCharsPerToken)*2 + 512
	ceiling := envInt("OLLAMA_NUM_PREDICT_TEXT", defaultTextNumCtxCeiling)
	return clamp(est, envInt("OLLAMA_NUM_PREDICT_TEXT_MIN", defaultTextNumPredictMin), ceiling)
}

// visionEndpoint reports which endpoint per-page vision should use. Kept
// configurable so the bench can A/B chat against generate on one image.
func visionEndpoint() string {
	if os.Getenv("OLLAMA_VISION_ENDPOINT") == visionEndpointGenerate {
		return visionEndpointGenerate
	}
	return visionEndpointChat
}

func clamp(v, lo, hi int) int {
	if lo > hi {
		lo, hi = hi, lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envIntMin(key string, def, min int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= min {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}
