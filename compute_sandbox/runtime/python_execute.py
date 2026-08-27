import io
import ast
import hashlib
import json
import keyword
import sys
import traceback
from contextlib import redirect_stdout, redirect_stderr

envelope = json.loads(sys.stdin.read() or "{}")
input_data = envelope.get("input") or {}
sdk_context = envelope.get("sdkContext") or {}
user_script = str(envelope.get("script") or "")
dependencies = envelope.get("dependencies") if isinstance(envelope.get("dependencies"), list) else []
argv = input_data.get("argv") if isinstance(input_data.get("argv"), list) else []
dp = sdk_context.get("variables") or {}

class DatapointSDK:
    def get(self, path):
        key = str(path or "")
        item = (sdk_context.get("datapoints") or {}).get(key)
        if item is None:
            raise RuntimeError("ctx.datapoint.get is not declared or prefetched: " + key)
        return item.get("value")

    def meta(self, path):
        key = str(path or "")
        item = (sdk_context.get("datapoints") or {}).get(key)
        if item is None:
            raise RuntimeError("ctx.datapoint.meta is not declared or prefetched: " + key)
        return item

class SQLSDK:
    def query(self, key, _params=None):
        name = str(key or "")
        sql_values = sdk_context.get("sql") or {}
        if name not in sql_values:
            raise RuntimeError("ctx.sql.query is not declared or prefetched: " + name)
        return sql_values[name]

class ComputeSDK:
    def __init__(self):
        self.datapoint = DatapointSDK()
        self.sql = SQLSDK()

ctx = ComputeSDK()

allowed_imports = set()
for item in dependencies:
    if not isinstance(item, dict):
        continue
    package_name = str(item.get("packageName") or "")
    import_name = str(item.get("importName") or package_name)
    key = hashlib.sha256(package_name.lower().encode("utf-8")).hexdigest()[:24]
    sys.path.insert(0, "/dependencies/python/" + key)
    allowed_imports.add(import_name.split(".", 1)[0])

native_import = __import__
def safe_import(name, globals=None, locals=None, fromlist=(), level=0):
    root = str(name or "").split(".", 1)[0]
    if level != 0 or root not in allowed_imports:
        raise ImportError("依赖未在当前工程安装或未被脚本声明: " + str(name))
    return native_import(name, globals, locals, fromlist, level)

# 不向用户脚本提供 __import__、open、eval、exec、compile、globals、locals 等宿主入口。
safe_builtins = {
    "None": None, "True": True, "False": False,
	"__build_class__": __build_class__, "object": object,
    "abs": abs, "all": all, "any": any, "bool": bool, "dict": dict, "enumerate": enumerate,
    "float": float, "int": int, "isinstance": isinstance, "len": len, "list": list, "map": map,
    "max": max, "min": min, "next": next, "range": range, "reversed": reversed, "round": round,
    "set": set, "sorted": sorted, "str": str, "sum": sum, "tuple": tuple, "zip": zip,
    "Exception": Exception, "RuntimeError": RuntimeError, "TypeError": TypeError, "ValueError": ValueError,
	"__import__": safe_import,
}
scope = {"__builtins__": safe_builtins, "__name__": "__compute__", "argv": argv, "dp": dp, "ctx": ctx}
for name, value in dp.items() if isinstance(dp, dict) else []:
    if isinstance(name, str) and name.isidentifier() and not name.startswith("__") and not keyword.iskeyword(name):
        scope[name] = value

captured_out = io.StringIO()
captured_err = io.StringIO()
try:
    tree = ast.parse(user_script, "<compute>", "exec")
    blocked_names = {"__import__", "open", "eval", "exec", "compile", "globals", "locals", "vars", "getattr", "setattr", "delattr", "breakpoint", "help", "input"}
    for node in ast.walk(tree):
        if isinstance(node, (ast.Global, ast.Nonlocal)):
            raise RuntimeError("Python 沙箱不允许修改外层作用域")
        if isinstance(node, ast.Import):
            for alias in node.names:
                if alias.name.split(".", 1)[0] not in allowed_imports:
                    raise RuntimeError("依赖未在当前工程安装或未被脚本声明: " + alias.name)
        if isinstance(node, ast.ImportFrom):
            root = str(node.module or "").split(".", 1)[0]
            if node.level != 0 or root not in allowed_imports:
                raise RuntimeError("依赖未在当前工程安装或未被脚本声明: " + str(node.module or ""))
        if isinstance(node, ast.Attribute) and node.attr.startswith("__"):
            raise RuntimeError("Python 沙箱不允许访问双下划线反射属性")
        if isinstance(node, ast.Name) and (node.id.startswith("__") or node.id in blocked_names):
            raise RuntimeError("Python 沙箱不允许访问宿主入口: " + node.id)
    code = compile(tree, "<compute>", "exec")
    with redirect_stdout(captured_out), redirect_stderr(captured_err):
        exec(code, scope, scope)
        main = scope.get("main")
        if not callable(main):
            raise RuntimeError("python 脚本必须定义 main(argv, dp, ctx) 函数")
        output = main(argv, dp, ctx)
    logs = [line for line in captured_out.getvalue().splitlines() if line]
    sys.stdout.write(json.dumps({"output": output, "sideEffects": [], "logs": logs}, ensure_ascii=False))
except Exception:
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
