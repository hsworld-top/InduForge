import json
import ast
import sys

envelope = json.loads(sys.stdin.read() or "{}")
script = str(envelope.get("script") or "")
try:
    tree = ast.parse(script, "<compute>", "exec")
    blocked_names = {"__import__", "open", "eval", "exec", "compile", "globals", "locals", "vars", "getattr", "setattr", "delattr", "breakpoint", "help", "input"}
    for node in ast.walk(tree):
        if isinstance(node, (ast.Global, ast.Nonlocal)):
            raise SyntaxError("Python 沙箱不允许修改外层作用域")
        if isinstance(node, ast.Attribute) and node.attr.startswith("__"):
            raise SyntaxError("Python 沙箱不允许访问双下划线反射属性")
        if isinstance(node, ast.Name) and (node.id.startswith("__") or node.id in blocked_names):
            raise SyntaxError("Python 沙箱不允许访问宿主入口: " + node.id)
    compile(tree, "<compute>", "exec")
    sys.stdout.write(json.dumps({"diagnostics": []}))
except SyntaxError as error:
    line = int(error.lineno or 1)
    column = int(error.offset or 1)
    end_line = int(getattr(error, "end_lineno", None) or line)
    end_column = int(getattr(error, "end_offset", None) or column + 1)
    sys.stdout.write(json.dumps({"diagnostics": [{"severity": "error", "message": str(error.msg or error), "line": max(1, line), "column": max(1, column), "endLine": max(1, end_line), "endColumn": max(2, end_column), "source": "python"}]}))
