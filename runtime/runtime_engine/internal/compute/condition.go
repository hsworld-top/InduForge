package compute

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math/big"
	"strconv"

	"github.com/indu-forge/runtime-engine/internal/model"
)

// EvaluateCondition is a deliberately closed expression evaluator.  It admits
// literals, input aliases, arithmetic, comparison and boolean operators only;
// calls, selectors, indexing, assignments and reflection have no AST path.
func EvaluateCondition(source string, values map[string]json.RawMessage) (bool, error) {
	expr, err := parser.ParseExpr(source)
	if err != nil {
		return false, errors.New("condition 表达式语法无效")
	}
	value, err := evaluateConditionNode(expr, values)
	if err != nil {
		return false, err
	}
	result, ok := value.(bool)
	if !ok {
		return false, errors.New("condition 表达式必须返回 bool")
	}
	return result, nil
}

func evaluateConditionNode(node ast.Expr, values map[string]json.RawMessage) (any, error) {
	switch n := node.(type) {
	case *ast.Ident:
		if n.Name == "true" {
			return true, nil
		}
		if n.Name == "false" {
			return false, nil
		}
		raw, ok := values[n.Name]
		if !ok {
			return nil, fmt.Errorf("condition 缺少变量 %s", n.Name)
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil || decoder.Decode(&struct{}{}) != io.EOF || !finiteConditionValue(value) {
			return nil, errors.New("condition 输入非法")
		}
		if number, ok := value.(json.Number); ok {
			exact, valid := ratFromNumber(number.String())
			if !valid {
				return nil, errors.New("condition 输入非法")
			}
			return exact, nil
		}
		return value, nil
	case *ast.BasicLit:
		switch n.Kind {
		case token.INT, token.FLOAT:
			value, ok := ratFromNumber(n.Value)
			if !ok {
				return nil, errors.New("condition 数字非法")
			}
			return value, nil
		case token.STRING:
			return strconv.Unquote(n.Value)
		}
	case *ast.ParenExpr:
		return evaluateConditionNode(n.X, values)
	case *ast.UnaryExpr:
		value, err := evaluateConditionNode(n.X, values)
		if err != nil {
			return nil, err
		}
		if n.Op == token.NOT {
			b, ok := value.(bool)
			if !ok {
				return nil, errors.New("! 仅支持 bool")
			}
			return !b, nil
		}
		f, ok := conditionNumber(value)
		if !ok || (n.Op != token.ADD && n.Op != token.SUB) {
			return nil, errors.New("一元运算非法")
		}
		if n.Op == token.SUB {
			return new(big.Rat).Neg(f), nil
		}
		return f, nil
	case *ast.BinaryExpr:
		left, err := evaluateConditionNode(n.X, values)
		if err != nil {
			return nil, err
		}
		if n.Op == token.LAND {
			b, ok := left.(bool)
			if !ok {
				return nil, errors.New("&& 仅支持 bool")
			}
			if !b {
				return false, nil
			}
			right, err := evaluateConditionNode(n.Y, values)
			if err != nil {
				return nil, err
			}
			b, ok = right.(bool)
			if !ok {
				return nil, errors.New("&& 仅支持 bool")
			}
			return b, nil
		}
		if n.Op == token.LOR {
			b, ok := left.(bool)
			if !ok {
				return nil, errors.New("|| 仅支持 bool")
			}
			if b {
				return true, nil
			}
			right, err := evaluateConditionNode(n.Y, values)
			if err != nil {
				return nil, err
			}
			b, ok = right.(bool)
			if !ok {
				return nil, errors.New("|| 仅支持 bool")
			}
			return b, nil
		}
		right, err := evaluateConditionNode(n.Y, values)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case token.EQL, token.NEQ:
			leftNumber, leftNumeric := conditionNumber(left)
			rightNumber, rightNumeric := conditionNumber(right)
			equal := leftNumeric && rightNumeric && leftNumber.Cmp(rightNumber) == 0
			if !leftNumeric || !rightNumeric {
				leftRaw, _ := json.Marshal(left)
				rightRaw, _ := json.Marshal(right)
				equal = bytes.Equal(leftRaw, rightRaw)
			}
			if n.Op == token.NEQ {
				equal = !equal
			}
			return equal, nil
		case token.ADD, token.SUB, token.MUL, token.QUO:
			a, aOK := conditionNumber(left)
			b, bOK := conditionNumber(right)
			if !aOK || !bOK || (n.Op == token.QUO && b.Sign() == 0) {
				return nil, errors.New("算术仅支持有限数字")
			}
			out := new(big.Rat)
			switch n.Op {
			case token.ADD:
				out.Add(a, b)
			case token.SUB:
				out.Sub(a, b)
			case token.MUL:
				out.Mul(a, b)
			default:
				out.Quo(a, b)
			}
			return out, nil
		case token.GTR, token.GEQ, token.LSS, token.LEQ:
			if a, ok := conditionNumber(left); ok {
				b, ok := conditionNumber(right)
				if !ok {
					return nil, errors.New("比较类型不一致")
				}
				compared := a.Cmp(b)
				switch n.Op {
				case token.GTR:
					return compared > 0, nil
				case token.GEQ:
					return compared >= 0, nil
				case token.LSS:
					return compared < 0, nil
				default:
					return compared <= 0, nil
				}
			}
			a, aOK := left.(string)
			b, bOK := right.(string)
			if !aOK || !bOK {
				return nil, errors.New("比较仅支持数字或文本")
			}
			switch n.Op {
			case token.GTR:
				return a > b, nil
			case token.GEQ:
				return a >= b, nil
			case token.LSS:
				return a < b, nil
			default:
				return a <= b, nil
			}
		}
	}
	return nil, errors.New("condition 表达式包含不允许的语法")
}

func conditionNumber(value any) (*big.Rat, bool) {
	switch v := value.(type) {
	case *big.Rat:
		return new(big.Rat).Set(v), true
	case json.Number:
		return ratFromNumber(v.String())
	}
	return nil, false
}

// ratFromNumber shares the model-layer decimal parser so condition literals
// and datapoint values have identical integer and high-precision semantics.
func ratFromNumber(value string) (*big.Rat, bool) {
	return model.ParseNonNegativeOrSignedNumber(json.RawMessage(value))
}

func finiteConditionValue(value any) bool {
	switch v := value.(type) {
	case json.Number:
		_, ok := ratFromNumber(v.String())
		return ok
	case []any:
		for _, x := range v {
			if !finiteConditionValue(x) {
				return false
			}
		}
	case map[string]any:
		for _, x := range v {
			if !finiteConditionValue(x) {
				return false
			}
		}
	}
	return true
}
