package service

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
)

// 组合报警表达式采用受限的 Go 风格表达式。只允许输入别名、字面量、比较、布尔和基础算术，
// 不允许函数、字段访问或索引，避免开发态配置隐式获得执行能力。
var alarmInputKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateDerivedAlarmExpression(expression string, allowedAliases map[string]bool) error {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return badAlarm("组合报警表达式语法无效")
	}
	if err = validateAlarmExpressionNode(expr, allowedAliases); err != nil {
		return err
	}
	return nil
}

func validateAlarmExpressionNode(node ast.Expr, aliases map[string]bool) error {
	switch value := node.(type) {
	case *ast.Ident:
		if value.Name == "true" || value.Name == "false" || aliases[value.Name] {
			return nil
		}
	case *ast.BasicLit:
		if value.Kind == token.INT || value.Kind == token.FLOAT || value.Kind == token.STRING || value.Kind == token.CHAR {
			return nil
		}
	case *ast.ParenExpr:
		return validateAlarmExpressionNode(value.X, aliases)
	case *ast.UnaryExpr:
		if value.Op == token.NOT || value.Op == token.ADD || value.Op == token.SUB {
			return validateAlarmExpressionNode(value.X, aliases)
		}
	case *ast.BinaryExpr:
		switch value.Op {
		case token.LAND, token.LOR, token.EQL, token.NEQ, token.GTR, token.GEQ, token.LSS, token.LEQ, token.ADD, token.SUB, token.MUL, token.QUO:
			if err := validateAlarmExpressionNode(value.X, aliases); err != nil {
				return err
			}
			return validateAlarmExpressionNode(value.Y, aliases)
		}
	}
	return badAlarm("组合报警表达式只支持输入别名、字面量、比较、布尔和基础算术")
}

func evaluateDerivedAlarmExpression(expression string, values map[string]any) (any, error) {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return nil, badAlarm("组合报警表达式语法无效")
	}
	return evaluateAlarmExpressionNode(expr, values)
}

type alarmExpressionType string

const (
	alarmExpressionBoolean alarmExpressionType = "boolean"
	alarmExpressionNumber  alarmExpressionType = "number"
	alarmExpressionText    alarmExpressionType = "text"
)

// validateDerivedAlarmExpressionTypes 在保存期阻止显然无效的组合表达式，避免必须等节点运行后才暴露类型错误。
func validateDerivedAlarmExpressionTypes(expression string, inputs []AlarmItemInput, condition AlarmCondition) error {
	aliases := make(map[string]alarmExpressionType, len(inputs))
	for _, input := range inputs {
		switch alarmDataCategory(input.DataType) {
		case "number":
			aliases[input.InputKey] = alarmExpressionNumber
		case "boolean":
			aliases[input.InputKey] = alarmExpressionBoolean
		case "text":
			aliases[input.InputKey] = alarmExpressionText
		default:
			return badAlarm("组合报警不支持结构化数据点作为表达式输入")
		}
	}
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return badAlarm("组合报警表达式语法无效")
	}
	actual, err := inferAlarmExpressionType(expr, aliases)
	if err != nil {
		return err
	}
	expected := alarmExpressionType("")
	switch condition.Kind {
	case "threshold", "range", "rate_of_change", "deviation":
		expected = alarmExpressionNumber
	case "state", "transition":
		expected = alarmExpressionBoolean
	case "text_match":
		expected = alarmExpressionText
	}
	if expected != "" && actual != expected {
		return badAlarm("组合表达式结果类型与报警条件不匹配")
	}
	return nil
}

func inferAlarmExpressionType(node ast.Expr, aliases map[string]alarmExpressionType) (alarmExpressionType, error) {
	require := func(actual, expected alarmExpressionType, message string) error {
		if actual != expected {
			return badAlarm(message)
		}
		return nil
	}
	switch value := node.(type) {
	case *ast.Ident:
		if value.Name == "true" || value.Name == "false" {
			return alarmExpressionBoolean, nil
		}
		kind, ok := aliases[value.Name]
		if !ok {
			return "", badAlarm("组合报警表达式引用了未知输入别名")
		}
		return kind, nil
	case *ast.BasicLit:
		switch value.Kind {
		case token.INT, token.FLOAT:
			return alarmExpressionNumber, nil
		case token.STRING, token.CHAR:
			return alarmExpressionText, nil
		}
	case *ast.ParenExpr:
		return inferAlarmExpressionType(value.X, aliases)
	case *ast.UnaryExpr:
		operand, err := inferAlarmExpressionType(value.X, aliases)
		if err != nil {
			return "", err
		}
		if value.Op == token.NOT {
			return alarmExpressionBoolean, require(operand, alarmExpressionBoolean, "组合表达式的 ! 只支持布尔输入")
		}
		if value.Op == token.ADD || value.Op == token.SUB {
			return alarmExpressionNumber, require(operand, alarmExpressionNumber, "组合表达式的一元运算只支持数值输入")
		}
	case *ast.BinaryExpr:
		left, err := inferAlarmExpressionType(value.X, aliases)
		if err != nil {
			return "", err
		}
		right, err := inferAlarmExpressionType(value.Y, aliases)
		if err != nil {
			return "", err
		}
		switch value.Op {
		case token.LAND, token.LOR:
			if err = require(left, alarmExpressionBoolean, "组合表达式的布尔运算只支持布尔输入"); err != nil {
				return "", err
			}
			return alarmExpressionBoolean, require(right, alarmExpressionBoolean, "组合表达式的布尔运算只支持布尔输入")
		case token.ADD, token.SUB, token.MUL, token.QUO:
			if err = require(left, alarmExpressionNumber, "组合表达式的算术运算只支持数值输入"); err != nil {
				return "", err
			}
			return alarmExpressionNumber, require(right, alarmExpressionNumber, "组合表达式的算术运算只支持数值输入")
		case token.EQL, token.NEQ:
			if left != right {
				return "", badAlarm("组合表达式的相等比较两侧类型必须一致")
			}
			return alarmExpressionBoolean, nil
		case token.GTR, token.GEQ, token.LSS, token.LEQ:
			if left != right || (left != alarmExpressionNumber && left != alarmExpressionText) {
				return "", badAlarm("组合表达式的大小比较两侧必须同为数值或文本")
			}
			return alarmExpressionBoolean, nil
		}
	}
	return "", badAlarm("组合报警表达式无法识别结果类型")
}

func evaluateAlarmExpressionNode(node ast.Expr, values map[string]any) (any, error) {
	switch value := node.(type) {
	case *ast.Ident:
		if value.Name == "true" {
			return true, nil
		}
		if value.Name == "false" {
			return false, nil
		}
		resolved, ok := values[value.Name]
		if !ok {
			return nil, badAlarm(fmt.Sprintf("组合试算缺少输入别名 %s 的值", value.Name))
		}
		return resolved, nil
	case *ast.BasicLit:
		switch value.Kind {
		case token.INT, token.FLOAT:
			return strconv.ParseFloat(value.Value, 64)
		case token.STRING, token.CHAR:
			return strconv.Unquote(value.Value)
		}
	case *ast.ParenExpr:
		return evaluateAlarmExpressionNode(value.X, values)
	case *ast.UnaryExpr:
		operand, err := evaluateAlarmExpressionNode(value.X, values)
		if err != nil {
			return nil, err
		}
		switch value.Op {
		case token.NOT:
			boolean, ok := operand.(bool)
			if !ok {
				return nil, badAlarm("组合表达式的 ! 只支持布尔值")
			}
			return !boolean, nil
		case token.ADD, token.SUB:
			number, ok := anyFloat(operand)
			if !ok {
				return nil, badAlarm("组合表达式的一元运算只支持数值")
			}
			if value.Op == token.SUB {
				return -number, nil
			}
			return number, nil
		}
	case *ast.BinaryExpr:
		left, err := evaluateAlarmExpressionNode(value.X, values)
		if err != nil {
			return nil, err
		}
		if value.Op == token.LAND {
			leftBool, ok := left.(bool)
			if !ok {
				return nil, badAlarm("组合表达式的 && 只支持布尔值")
			}
			if !leftBool {
				return false, nil
			}
			right, rightErr := evaluateAlarmExpressionNode(value.Y, values)
			if rightErr != nil {
				return nil, rightErr
			}
			rightBool, ok := right.(bool)
			if !ok {
				return nil, badAlarm("组合表达式的 && 只支持布尔值")
			}
			return rightBool, nil
		}
		if value.Op == token.LOR {
			leftBool, ok := left.(bool)
			if !ok {
				return nil, badAlarm("组合表达式的 || 只支持布尔值")
			}
			if leftBool {
				return true, nil
			}
			right, rightErr := evaluateAlarmExpressionNode(value.Y, values)
			if rightErr != nil {
				return nil, rightErr
			}
			rightBool, ok := right.(bool)
			if !ok {
				return nil, badAlarm("组合表达式的 || 只支持布尔值")
			}
			return rightBool, nil
		}
		right, err := evaluateAlarmExpressionNode(value.Y, values)
		if err != nil {
			return nil, err
		}
		switch value.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO:
			leftNumber, leftOK := anyFloat(left)
			rightNumber, rightOK := anyFloat(right)
			if !leftOK || !rightOK || (value.Op == token.QUO && rightNumber == 0) {
				return nil, badAlarm("组合表达式的算术运算只支持非零数值除数")
			}
			switch value.Op {
			case token.ADD:
				return leftNumber + rightNumber, nil
			case token.SUB:
				return leftNumber - rightNumber, nil
			case token.MUL:
				return leftNumber * rightNumber, nil
			default:
				return leftNumber / rightNumber, nil
			}
		case token.EQL, token.NEQ:
			equal := fmt.Sprint(left) == fmt.Sprint(right)
			if value.Op == token.NEQ {
				equal = !equal
			}
			return equal, nil
		case token.GTR, token.GEQ, token.LSS, token.LEQ:
			leftNumber, leftOK := anyFloat(left)
			rightNumber, rightOK := anyFloat(right)
			if leftOK && rightOK {
				switch value.Op {
				case token.GTR:
					return leftNumber > rightNumber, nil
				case token.GEQ:
					return leftNumber >= rightNumber, nil
				case token.LSS:
					return leftNumber < rightNumber, nil
				default:
					return leftNumber <= rightNumber, nil
				}
			}
			leftText, leftString := left.(string)
			rightText, rightString := right.(string)
			if !leftString || !rightString {
				return nil, badAlarm("组合表达式的比较两侧必须同为数值或文本")
			}
			switch value.Op {
			case token.GTR:
				return leftText > rightText, nil
			case token.GEQ:
				return leftText >= rightText, nil
			case token.LSS:
				return leftText < rightText, nil
			default:
				return leftText <= rightText, nil
			}
		}
	}
	return nil, badAlarm("组合报警表达式无法计算")
}
