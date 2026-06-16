# ADR-0002: mapstructure for decoding

- Status: Accepted
- Context: Config arrives as nested `map[string]any` from YAML and env key splitting. Manual field coercion is error-prone.
- Decision: Decode into Go structs with `github.com/go-viper/mapstructure/v2` via go-config codecs. Custom hooks only for `time.Duration` strings.
- Consequences: Struct tags and nested field names must stay aligned with YAML keys. Prefer mapstructure over hand-written conversion loops.
