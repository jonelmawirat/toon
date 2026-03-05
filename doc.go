// Package toon implements TOON (Token-Oriented Object Notation), a
// line-oriented and indentation-based text format for JSON-like data.
//
// Core APIs:
//   - Marshal / Unmarshal for one-shot conversion
//   - Encoder / Decoder for streaming workflows
//
// Decoded values are returned as Value. Use AsObject and AsArray to safely
// read object and array values, including optimized internal representations
// used by strict tabular decoding.
package toon
