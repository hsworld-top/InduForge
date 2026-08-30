package model

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// ValidateDataPointValue is shared by artifact validation and compute output
// validation. It accepts exactly one finite JSON value and never routes a
// number through float64 before integer/decimal checks.
func ValidateDataPointValue(raw json.RawMessage, dataType string, allowNull bool) error {
	value, ok := strictJSONValue(raw)
	if !ok {
		return errValue
	}
	if value == nil {
		if allowNull {
			return nil
		}
		return errValue
	}
	switch dataType {
	case "bool":
		_, ok := value.(bool)
		if ok {
			return nil
		}
	case "string":
		_, ok := value.(string)
		if ok {
			return nil
		}
	case "bytes":
		text, ok := value.(string)
		if ok {
			decoded, err := base64.StdEncoding.DecodeString(text)
			// DecodeString accepts CR/LF and a few non-canonical encodings;
			// V1 bytes wire values are canonical padded standard base64 only.
			if err == nil && base64.StdEncoding.EncodeToString(decoded) == text {
				return nil
			}
		}
	case "datetime":
		text, ok := value.(string)
		if ok && strings.HasSuffix(text, "Z") {
			if _, err := time.Parse(time.RFC3339Nano, text); err == nil {
				return nil
			}
		}
	case "object":
		if _, ok := value.(map[string]any); ok {
			return nil
		}
	case "array":
		if _, ok := value.([]any); ok {
			return nil
		}
	default:
		number, ok := value.(json.Number)
		if !ok {
			return errValue
		}
		rat, ok := ParseNonNegativeOrSignedNumber([]byte(number.String()))
		if !ok {
			return errValue
		}
		if dataType == "decimal" {
			return nil
		}
		if dataType == "float32" {
			max, _ := ParseNonNegativeOrSignedNumber([]byte("3.40282346638528859811704183484516925440e38"))
			if new(big.Rat).Abs(rat).Cmp(max) <= 0 {
				return nil
			}
			return errValue
		}
		if dataType == "float64" {
			f, _ := new(big.Float).SetRat(rat).Float64()
			if !math.IsInf(f, 0) && !math.IsNaN(f) {
				return nil
			}
			return errValue
		}
		if integerDataType(rat, dataType) {
			return nil
		}
	}
	return errValue
}

var errValue = valueError{}

type valueError struct{}

func (valueError) Error() string { return "数据类型值非法" }

func strictJSONValue(raw json.RawMessage) (any, bool) {
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&value) != nil || d.Decode(&struct{}{}) != io.EOF {
		return nil, false
	}
	return value, true
}
func ParseNonNegativeNumber(raw json.RawMessage) (*big.Rat, bool) {
	rat, ok := ParseNonNegativeOrSignedNumber(raw)
	return rat, ok && rat.Sign() >= 0
}
func ParseNonNegativeOrSignedNumber(raw json.RawMessage) (*big.Rat, bool) {
	value, ok := strictJSONValue(raw)
	if !ok {
		return nil, false
	}
	number, ok := value.(json.Number)
	if !ok {
		return nil, false
	}
	return parseExactNumber(number.String())
}
func parseExactNumber(value string) (*big.Rat, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "NaInF") {
		return nil, false
	}
	sign := 1
	if value[0] == '-' {
		sign = -1
		value = value[1:]
	} else if value[0] == '+' {
		return nil, false
	}
	parts := strings.Split(strings.ToLower(value), "e")
	if len(parts) > 2 {
		return nil, false
	}
	exponent := 0
	if len(parts) == 2 {
		var err error
		exponent, err = strconv.Atoi(parts[1])
		if err != nil || exponent > 10000 || exponent < -10000 {
			return nil, false
		}
	}
	dot := strings.Split(parts[0], ".")
	if len(dot) > 2 || dot[0] == "" {
		return nil, false
	}
	whole, frac := dot[0], ""
	if len(dot) == 2 {
		frac = dot[1]
		if frac == "" {
			return nil, false
		}
	}
	if (whole != "0" && (whole[0] == '0' || !decimalDigits(whole))) || !decimalDigits(frac) {
		return nil, false
	}
	joined := strings.TrimLeft(whole+frac, "0")
	if joined == "" {
		return new(big.Rat), true
	}
	num := new(big.Int)
	if _, ok := num.SetString(joined, 10); !ok {
		return nil, false
	}
	if sign < 0 {
		num.Neg(num)
	}
	scale := len(frac) - exponent
	den := big.NewInt(1)
	if scale >= 0 {
		den.Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	} else {
		num.Mul(num, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-scale)), nil))
	}
	return new(big.Rat).SetFrac(num, den), true
}
func decimalDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func integerDataType(value *big.Rat, typ string) bool {
	if !value.IsInt() {
		return false
	}
	integer := value.Num()
	bits, ok := map[string]uint{"int8": 8, "uint8": 8, "int16": 16, "uint16": 16, "int32": 32, "uint32": 32, "int64": 64, "uint64": 64}[typ]
	if !ok {
		return false
	}
	if strings.HasPrefix(typ, "u") {
		return integer.Sign() >= 0 && integer.BitLen() <= int(bits)
	}
	limit := new(big.Int).Lsh(big.NewInt(1), bits-1)
	return integer.Cmp(new(big.Int).Neg(limit)) >= 0 && integer.Cmp(new(big.Int).Sub(limit, big.NewInt(1))) <= 0
}
