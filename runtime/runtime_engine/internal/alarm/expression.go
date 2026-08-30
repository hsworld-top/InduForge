package alarm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math"
	"math/big"
	"strconv"
)

// evalExpression 是封闭、确定性的组合报警表达式：无函数、字段、索引、反射或 I/O。
func evalExpression(source string, values map[string]json.RawMessage) (json.RawMessage, error) {
	expr, err := parser.ParseExpr(source)
	if err != nil {
		return nil, errors.New("derived expression 语法无效")
	}
	value, err := evalNode(expr, values)
	if err != nil {
		return nil, err
	}
	raw, err := marshalExpressionValue(value)
	if err != nil || !finiteRaw(raw) {
		return nil, errors.New("derived expression 结果非法")
	}
	return raw, nil
}

// validateExpression 在加载期拒绝未知 alias 与所有非封闭语法；求值期只做数据相关类型判断。
func validateExpression(source string, aliases map[string]bool) error {
	expr, err := parser.ParseExpr(source)
	if err != nil {
		return errors.New("derived expression 语法无效")
	}
	valid := true
	ast.Inspect(expr, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		switch n := node.(type) {
		case *ast.Ident:
			if n.Name != "true" && n.Name != "false" && !aliases[n.Name] {
				valid = false
			}
		case *ast.CallExpr, *ast.SelectorExpr, *ast.IndexExpr, *ast.SliceExpr, *ast.CompositeLit, *ast.KeyValueExpr:
			valid = false
		case *ast.UnaryExpr:
			if n.Op != token.NOT && n.Op != token.ADD && n.Op != token.SUB {
				valid = false
			}
		case *ast.BinaryExpr:
			switch n.Op {
			case token.LAND, token.LOR, token.EQL, token.NEQ, token.GTR, token.GEQ, token.LSS, token.LEQ, token.ADD, token.SUB, token.MUL, token.QUO:
			default:
				valid = false
			}
		case *ast.BasicLit, *ast.ParenExpr:
		default:
			valid = false
		}
		return valid
	})
	if !valid {
		return errors.New("derived expression 包含不允许的语法或 alias")
	}
	return nil
}

func evalNode(node ast.Expr, values map[string]json.RawMessage) (any, error) {
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
			return nil, fmt.Errorf("derived expression 缺少输入 %s", n.Name)
		}
		var v any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if d.Decode(&v) != nil || !finiteAny(v) {
			return nil, errors.New("derived expression 输入非法")
		}
		if n, ok := v.(json.Number); ok {
			r, ok := new(big.Rat).SetString(n.String())
			if !ok {
				return nil, errors.New("derived expression 数字非法")
			}
			return r, nil
		}
		return v, nil
	case *ast.BasicLit:
		switch n.Kind {
		case token.INT, token.FLOAT:
			r, ok := new(big.Rat).SetString(n.Value)
			if !ok {
				return nil, errors.New("非有限数字")
			}
			return r, nil
		case token.STRING, token.CHAR:
			return strconv.Unquote(n.Value)
		}
	case *ast.ParenExpr:
		return evalNode(n.X, values)
	case *ast.UnaryExpr:
		v, e := evalNode(n.X, values)
		if e != nil {
			return nil, e
		}
		f, ok := ratAny(v)
		if n.Op == token.NOT {
			b, ok := v.(bool)
			if !ok {
				return nil, errors.New("! 仅支持布尔")
			}
			return !b, nil
		}
		if !ok || (n.Op != token.ADD && n.Op != token.SUB) {
			return nil, errors.New("一元运算非法")
		}
		if n.Op == token.SUB {
			return new(big.Rat).Neg(f), nil
		}
		return f, nil
	case *ast.BinaryExpr:
		left, e := evalNode(n.X, values)
		if e != nil {
			return nil, e
		}
		if n.Op == token.LAND {
			b, ok := left.(bool)
			if !ok {
				return nil, errors.New("&& 仅支持布尔")
			}
			if !b {
				return false, nil
			}
			right, e := evalNode(n.Y, values)
			if e != nil {
				return nil, e
			}
			b, ok = right.(bool)
			if !ok {
				return nil, errors.New("&& 仅支持布尔")
			}
			return b, nil
		}
		if n.Op == token.LOR {
			b, ok := left.(bool)
			if !ok {
				return nil, errors.New("|| 仅支持布尔")
			}
			if b {
				return true, nil
			}
			right, e := evalNode(n.Y, values)
			if e != nil {
				return nil, e
			}
			b, ok = right.(bool)
			if !ok {
				return nil, errors.New("|| 仅支持布尔")
			}
			return b, nil
		}
		right, e := evalNode(n.Y, values)
		if e != nil {
			return nil, e
		}
		if n.Op == token.EQL || n.Op == token.NEQ {
			equal := equalValue(left, right)
			if n.Op == token.NEQ {
				equal = !equal
			}
			return equal, nil
		}
		if n.Op == token.ADD || n.Op == token.SUB || n.Op == token.MUL || n.Op == token.QUO {
			a, aok := ratAny(left)
			b, bok := ratAny(right)
			if !aok || !bok || (n.Op == token.QUO && b.Sign() == 0) {
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
		}
		if n.Op == token.GTR || n.Op == token.GEQ || n.Op == token.LSS || n.Op == token.LEQ {
			if a, aok := ratAny(left); aok {
				b, bok := ratAny(right)
				if !bok {
					return nil, errors.New("比较类型不一致")
				}
				return compareRat(n.Op, a, b), nil
			}
			a, aok := left.(string)
			b, bok := right.(string)
			if !aok || !bok {
				return nil, errors.New("比较仅支持数字或文本")
			}
			return compare(n.Op, a, b), nil
		}
	}
	return nil, errors.New("derived expression 包含不允许的语法")
}

func compare[T ~float64 | ~string](op token.Token, a, b T) bool {
	switch op {
	case token.GTR:
		return a > b
	case token.GEQ:
		return a >= b
	case token.LSS:
		return a < b
	default:
		return a <= b
	}
}
func equalValue(a, b any) bool {
	if left, ok := ratAny(a); ok {
		right, rok := ratAny(b)
		return rok && left.Cmp(right) == 0
	}
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return string(ra) == string(rb)
}
func compareRat(op token.Token, a, b *big.Rat) bool {
	c := a.Cmp(b)
	switch op {
	case token.GTR:
		return c > 0
	case token.GEQ:
		return c >= 0
	case token.LSS:
		return c < 0
	default:
		return c <= 0
	}
}
func marshalExpressionValue(value any) (json.RawMessage, error) {
	if r, ok := value.(*big.Rat); ok {
		return decimalJSON(r)
	}
	return json.Marshal(value)
}
func decimalJSON(r *big.Rat) (json.RawMessage, error) {
	den := new(big.Int).Set(r.Denom())
	two := big.NewInt(2)
	five := big.NewInt(5)
	digits := 0
	for new(big.Int).Mod(den, two).Sign() == 0 {
		den.Div(den, two)
		digits++
	}
	fives := 0
	for new(big.Int).Mod(den, five).Sign() == 0 {
		den.Div(den, five)
		fives++
	}
	if den.Cmp(big.NewInt(1)) != 0 {
		return nil, errors.New("derived division 结果不能精确表示为 JSON decimal")
	}
	if fives > digits {
		digits = fives
	}
	return json.RawMessage(r.FloatString(digits)), nil
}
func finiteRaw(raw []byte) bool {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&v) != nil || !finiteAny(v) {
		return false
	}
	var trailing any
	return d.Decode(&trailing) == io.EOF
}
func finiteAny(v any) bool {
	switch x := v.(type) {
	case json.Number:
		_, ok := ratAny(x)
		return ok
	case float64:
		return !math.IsNaN(x) && !math.IsInf(x, 0)
	case []any:
		for _, e := range x {
			if !finiteAny(e) {
				return false
			}
		}
	case map[string]any:
		for _, e := range x {
			if !finiteAny(e) {
				return false
			}
		}
	}
	return true
}
