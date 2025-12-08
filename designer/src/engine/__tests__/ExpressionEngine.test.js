/**
 * ExpressionEngine 单元测试
 */
import { describe, it, expect } from 'vitest'
import expressionEngine from '../binding/ExpressionEngine'

describe('ExpressionEngine', () => {
  it('should evaluate simple expressions', () => {
    expressionEngine.setContext({ vars: { count: 10 } })
    expect(expressionEngine.evaluate('{{ vars.count * 2 }}')).toBe(20)
  })
  
  it('should evaluate string concatenation', () => {
    expressionEngine.setContext({ vars: { name: 'World' } })
    expect(expressionEngine.evaluate('Hello {{ vars.name }}!')).toBe('Hello World!')
  })
  
  it('should use built-in format function', () => {
    expressionEngine.setContext({ vars: { value: 3.14159 } })
    expect(expressionEngine.evaluate('{{ format(vars.value, 2) }}')).toBe('3.14')
  })
  
  it('should use built-in if function', () => {
    expressionEngine.setContext({ vars: { status: 1 } })
    expect(expressionEngine.evaluate('{{ if(vars.status == 1, "运行", "停止") }}')).toBe('运行')
  })
  
  it('should handle array operations', () => {
    expressionEngine.setContext({ 
      data: { 
        list: [{ value: 10 }, { value: 20 }, { value: 30 }] 
      } 
    })
    expect(expressionEngine.evaluate('{{ sum(data.list, "value") }}')).toBe(60)
    expect(expressionEngine.evaluate('{{ avg(data.list, "value") }}')).toBe(20)
    expect(expressionEngine.evaluate('{{ max(data.list, "value") }}')).toBe(30)
    expect(expressionEngine.evaluate('{{ min(data.list, "value") }}')).toBe(10)
  })
  
  it('should return original value for non-expression strings', () => {
    expect(expressionEngine.evaluate('plain text')).toBe('plain text')
  })
  
  it('should handle null or undefined gracefully', () => {
    expect(expressionEngine.evaluate(null)).toBe(null)
    expect(expressionEngine.evaluate(undefined)).toBe(undefined)
  })
})
