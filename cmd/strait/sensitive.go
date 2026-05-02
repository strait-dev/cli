package main

import "encoding/json"

// sensitiveMask is the placeholder rendered in place of secret material when
// --reveal is not set.
const sensitiveMask = "********"

// maskString returns the sensitiveMask placeholder when reveal is false and
// the string is non-empty; otherwise returns s unchanged.
func maskString(s string, reveal bool) string {
	if reveal || s == "" {
		return s
	}
	return sensitiveMask
}

// maskRawJSON returns a JSON placeholder ("********") when reveal is false and
// the input is non-empty; otherwise returns raw unchanged. The resulting value
// is always valid JSON, so callers can embed it into a struct that gets passed
// to printData without breaking the encoder.
func maskRawJSON(raw json.RawMessage, reveal bool) json.RawMessage {
	if reveal || len(raw) == 0 {
		return raw
	}
	return json.RawMessage(`"` + sensitiveMask + `"`)
}

// maskMapValues returns a new map of the same shape with every value replaced
// by sensitiveMask when reveal is false; otherwise returns m unchanged. Empty
// values are preserved as empty strings rather than masked, since "" carries
// no secret content but masking it would be misleading.
func maskMapValues(m map[string]string, reveal bool) map[string]string {
	if reveal || len(m) == 0 {
		return m
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if v == "" {
			out[k] = ""
			continue
		}
		out[k] = sensitiveMask
	}
	return out
}
