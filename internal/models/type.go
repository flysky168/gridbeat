package models

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

type Kind uint8

const (
	KindInvalid Kind = iota
	KindBool
	KindUint16
	KindSint16
	KindUint32
	KindSint32
	KindSint64
	KindUint64
	KindFloat32
	KindFloat64
	KindString
	KindBytes
	KindBitMask
)

func (k Kind) String() string {
	switch k {
	case KindBool:
		return "bool"
	case KindSint16:
		return "sint16"
	case KindUint16:
		return "uint16"
	case KindSint32:
		return "sint64"
	case KindUint32:
		return "uint64"
	case KindSint64:
		return "sint64"
	case KindUint64:
		return "uint64"
	case KindFloat32:
		return "float32"
	case KindFloat64:
		return "float64"
	case KindString:
		return "string"
	case KindBitMask:
		return "bitmask"
	case KindBytes:
		return "bytes"
	default:
		return "invalid"
	}
}

func ParseKind(s string) (Kind, error) {
	switch s {
	case "bool":
		return KindBool, nil
	case "sint16":
		return KindSint64, nil
	case "uint16":
		return KindUint64, nil
	case "sint32":
		return KindSint64, nil
	case "uint32":
		return KindUint64, nil
	case "sint64":
		return KindSint64, nil
	case "uint64":
		return KindUint64, nil
	case "float32":
		return KindFloat32, nil
	case "float64":
		return KindFloat64, nil
	case "string":
		return KindString, nil
	case "bytes":
		return KindBytes, nil
	case "bitmask":
		return KindBitMask, nil
	default:
		return KindInvalid, fmt.Errorf("unknown kind: %q", s)
	}
}

// Scalar stores a typed value in raw bytes.
type Scalar struct {
	Kind Kind   `json:"kind"`
	Raw  []byte `json:"raw"` // raw bytes, format depends on Kind
}

func (s *Scalar) IsZero() bool { return s.Kind == KindInvalid }

func (s *Scalar) Reset() {
	s.Kind = KindInvalid
	s.Raw = nil
}

// ---- setters ----

func (s *Scalar) SetBool(v bool) {
	s.Kind = KindBool
	if v {
		s.Raw = []byte{1}
	} else {
		s.Raw = []byte{0}
	}
}

func (s *Scalar) SetSint16(v int16) {
	s.Kind = KindSint16
	s.Raw = make([]byte, 2)
	binary.LittleEndian.PutUint16(s.Raw, uint16(v))
}

func (s *Scalar) SetUint16(v uint16) {
	s.Kind = KindUint16
	s.Raw = make([]byte, 2)
	binary.LittleEndian.PutUint16(s.Raw, v)
}

func (s *Scalar) SetSint32(v int32) {
	s.Kind = KindSint32
	s.Raw = make([]byte, 4)
	binary.LittleEndian.PutUint32(s.Raw, uint32(v))
}

func (s *Scalar) SetUint32(v uint32) {
	s.Kind = KindUint32
	s.Raw = make([]byte, 4)
	binary.LittleEndian.PutUint32(s.Raw, v)
}

func (s *Scalar) SetSint64(v int64) {
	s.Kind = KindSint64
	s.Raw = make([]byte, 8)
	binary.LittleEndian.PutUint64(s.Raw, uint64(v))
}

func (s *Scalar) SetUint64(v uint64) {
	s.Kind = KindUint64
	s.Raw = make([]byte, 8)
	binary.LittleEndian.PutUint64(s.Raw, v)
}

func (s *Scalar) SetFloat32(v float32) {
	s.Kind = KindFloat32
	s.Raw = make([]byte, 4)
	binary.LittleEndian.PutUint32(s.Raw, math.Float32bits(v))
}

func (s *Scalar) SetFloat64(v float64) {
	s.Kind = KindFloat64
	s.Raw = make([]byte, 8)
	binary.LittleEndian.PutUint64(s.Raw, math.Float64bits(v))
}

func (s *Scalar) SetString(v string) {
	s.Kind = KindString
	s.Raw = []byte(v) // UTF-8
}

func (s *Scalar) SetBytes(v []byte) {
	s.Kind = KindBytes
	s.Raw = append([]byte(nil), v...) // copy
}

// ---- getters (type-safe) ----

func (s Scalar) Bool() (bool, error) {
	if s.Kind != KindBool {
		return false, fmt.Errorf("kind mismatch: want bool, got %s", s.Kind)
	}
	if len(s.Raw) != 1 {
		return false, fmt.Errorf("invalid bool raw length: %d", len(s.Raw))
	}
	return s.Raw[0] != 0, nil
}

func (s Scalar) Sint16() (int16, error) {
	if s.Kind != KindSint16 {
		return 0, fmt.Errorf("kind mismatch: want int16, got %s", s.Kind)
	}
	if len(s.Raw) != 2 {
		return 0, fmt.Errorf("invalid int16 raw length: %d", len(s.Raw))
	}
	return int16(binary.LittleEndian.Uint16(s.Raw)), nil
}

func (s Scalar) Uint16() (uint16, error) {
	if s.Kind != KindUint16 {
		return 0, fmt.Errorf("kind mismatch: want uint16, got %s", s.Kind)
	}
	if len(s.Raw) != 2 {
		return 0, fmt.Errorf("invalid uint16 raw length: %d", len(s.Raw))
	}
	return binary.LittleEndian.Uint16(s.Raw), nil
}

func (s Scalar) Sint32() (int32, error) {
	if s.Kind != KindSint64 {
		return 0, fmt.Errorf("kind mismatch: want int32, got %s", s.Kind)
	}
	if len(s.Raw) != 4 {
		return 0, fmt.Errorf("invalid int32 raw length: %d", len(s.Raw))
	}
	return int32(binary.LittleEndian.Uint32(s.Raw)), nil
}

func (s Scalar) Uint32() (uint32, error) {
	if s.Kind != KindUint32 {
		return 0, fmt.Errorf("kind mismatch: want uint32, got %s", s.Kind)
	}
	if len(s.Raw) != 4 {
		return 0, fmt.Errorf("invalid uint32 raw length: %d", len(s.Raw))
	}
	return binary.LittleEndian.Uint32(s.Raw), nil
}

func (s Scalar) Sint64() (int64, error) {
	if s.Kind != KindSint64 {
		return 0, fmt.Errorf("kind mismatch: want int64, got %s", s.Kind)
	}
	if len(s.Raw) != 8 {
		return 0, fmt.Errorf("invalid int64 raw length: %d", len(s.Raw))
	}
	return int64(binary.LittleEndian.Uint64(s.Raw)), nil
}

func (s Scalar) Uint64() (uint64, error) {
	if s.Kind != KindUint64 {
		return 0, fmt.Errorf("kind mismatch: want uint64, got %s", s.Kind)
	}
	if len(s.Raw) != 8 {
		return 0, fmt.Errorf("invalid uint64 raw length: %d", len(s.Raw))
	}
	return binary.LittleEndian.Uint64(s.Raw), nil
}

func (s Scalar) Float32() (float32, error) {
	if s.Kind != KindFloat64 {
		return 0, fmt.Errorf("kind mismatch: want float64, got %s", s.Kind)
	}
	if len(s.Raw) != 8 {
		return 0, fmt.Errorf("invalid float64 raw length: %d", len(s.Raw))
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(s.Raw)), nil
}

func (s Scalar) Float64() (float64, error) {
	if s.Kind != KindFloat64 {
		return 0, fmt.Errorf("kind mismatch: want float64, got %s", s.Kind)
	}
	if len(s.Raw) != 8 {
		return 0, fmt.Errorf("invalid float64 raw length: %d", len(s.Raw))
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(s.Raw)), nil
}

func (s Scalar) String() (string, error) {
	if s.Kind != KindString {
		return "", fmt.Errorf("kind mismatch: want string, got %s", s.Kind)
	}
	return string(s.Raw), nil
}

func (s Scalar) Bytes() ([]byte, error) {
	if s.Kind != KindBytes {
		return nil, fmt.Errorf("kind mismatch: want bytes, got %s", s.Kind)
	}
	return append([]byte(nil), s.Raw...), nil
}

// ---- DB support: store as BLOB/bytea via []byte ----
//
// 注意：DB 里通常需要两列：kind + value_raw。
// 这里的 Value/Scan 只管 raw bytes；Kind 你用另一列存 string/enum。

func (s Scalar) Value() (driver.Value, error) {
	// this satisfies driver.Valuer when Scalar is used directly as a column type
	return s.Raw, nil
}

func (s *Scalar) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		s.Reset()
		return nil
	case []byte:
		s.Raw = append([]byte(nil), v...)
		return nil
	case string:
		s.Raw = []byte(v)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Scalar.Raw", src)
	}
}

// ---- JSON support (optional) ----
// JSON 形态：{"kind":"int64","value":123} / {"kind":"bool","value":true} ...
// 不直接暴露 raw bytes，API 更友好。

type jsonScalar struct {
	Kind  string          `json:"kind"`
	Value json.RawMessage `json:"value"`
}

func (s Scalar) MarshalJSON() ([]byte, error) {
	// raw bytes 走 bytes base64 不直观，这里输出“kind + value”
	switch s.Kind {
	case KindBool:
		v, err := s.Bool()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"kind": "bool", "value": v})
	case KindSint64:
		v, err := s.Sint64()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"kind": "int64", "value": v})
	case KindUint64:
		v, err := s.Uint64()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"kind": "uint64", "value": v})
	case KindFloat32:
		v, err := s.Float32()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"kind": "float32", "value": v})
	case KindFloat64:
		v, err := s.Float64()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"kind": "float64", "value": v})
	case KindString:
		v, err := s.String()
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"kind": "string", "value": v})
	case KindBytes:
		// bytes 默认 json 会 base64，OK
		return json.Marshal(map[string]any{"kind": "bytes", "value": s.Raw})
	default:
		return []byte(`{"kind":"invalid","value":null}`), nil
	}
}

func (s *Scalar) UnmarshalJSON(b []byte) error {
	var js jsonScalar
	if err := json.Unmarshal(b, &js); err != nil {
		return err
	}
	k, err := ParseKind(js.Kind)
	if err != nil {
		return err
	}

	s.Kind = k
	switch k {
	case KindBool:
		var v bool
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetBool(v)
	case KindSint16:
		var v int16
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetSint16(v)
	case KindUint16:
		var v uint16
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetUint16(v)
	case KindSint32:
		var v int32
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetSint32(v)
	case KindUint32:
		var v uint32
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetUint32(v)
	case KindSint64:
		var v int64
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetSint64(v)
	case KindUint64:
		var v uint64
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetUint64(v)
	case KindFloat32:
		var v float32
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetFloat32(v)
	case KindFloat64:
		var v float64
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetFloat64(v)
	case KindString:
		var v string
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetString(v)
	case KindBytes:
		// bytes 在 JSON 中是 base64
		var v []byte
		if err := json.Unmarshal(js.Value, &v); err != nil {
			return err
		}
		s.SetBytes(v)
	default:
		s.Reset()
	}
	return nil
}

// Helper: compare raw bytes (optional)
func (s Scalar) Equal(other Scalar) bool {
	return s.Kind == other.Kind && bytes.Equal(s.Raw, other.Raw)
}

var ErrKindMismatch = errors.New("kind mismatch")
