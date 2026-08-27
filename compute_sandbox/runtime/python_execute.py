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
side_effects = []

class SDKResult(dict):
    def __init__(self, code, msg, data):
        super().__init__(code=code, msg=msg, data=data)

    def __getattr__(self, name):
        if name in self:
            return self[name]
        raise AttributeError(name)

def sdk_result(code, msg, data):
    return SDKResult(code, msg, data)

def unsupported(path, operation):
    return sdk_result(40031, "数据点 " + path + " 在当前开发环境不支持 " + operation, None)

def build_sample(snapshot):
    return {
        "path": snapshot.get("path") or "",
        "value": snapshot.get("value"),
        "quality": snapshot.get("quality") or "unknown",
        "timestamp": snapshot.get("timestamp"),
        "observedAt": snapshot.get("observedAt"),
        "sourceTimestamp": snapshot.get("sourceTimestamp"),
        "status": snapshot.get("status"),
    }

class DataPoint:
    def __init__(self, snapshot):
        self._snapshot = snapshot
        self.id = snapshot.get("id")
        self.ref = str(snapshot.get("path") or "")
        self.path = self.ref
        self.name = snapshot.get("name")
        self.displayName = snapshot.get("displayName") or self.name
        self.dataType = snapshot.get("dataType")
        self.schema = snapshot.get("schema")
        self.source = {"type": snapshot.get("sourceType"), "id": snapshot.get("sourceId")}
        self.status = snapshot.get("status")
        self.unit = snapshot.get("unit")
        self.precision = snapshot.get("precision")
        self.min = snapshot.get("min")
        self.max = snapshot.get("max")
        self.defaultValue = snapshot.get("defaultValue")
        self.tags = list(snapshot.get("tags") or [])
        self.attributes = dict(snapshot.get("attributes") or {})
        source_capabilities = snapshot.get("capabilities") or {}
        self.capabilities = {
            operation: source_capabilities.get(operation) is True
            for operation in ("get", "read", "peek", "set", "subscribe", "history", "refresh", "run", "execute", "publish")
        }

    def _read_snapshot(self, operation, data):
        if not self.capabilities.get(operation):
            return unsupported(self.path, operation)
        return sdk_result(0, "ok", data)

    def _effect(self, operation, payload=None):
        if not self.capabilities.get(operation):
            return unsupported(self.path, operation)
        item = {"domain": "point", "operation": operation, "path": self.path, "payload": payload}
        side_effects.append(item)
        return sdk_result(0, "开发态副作用已记录", item)

    def get(self, _options=None):
        return self._read_snapshot("get", self._snapshot.get("value"))

    def read(self, _options=None):
        return self._read_snapshot("read", build_sample(self._snapshot))

    def peek(self):
        return self._read_snapshot("peek", build_sample(self._snapshot))

    def set(self, value, _options=None):
        return self._effect("set", value)

    def subscribe(self, _handler=None, _options=None):
        return unsupported(self.path, "subscribe")

    def history(self, _query=None):
        return unsupported(self.path, "history")

    def refresh(self, options=None):
        return self._effect("refresh", options)

    def run(self, runtime_input=None):
        return self._effect("run", runtime_input)

    def execute(self, runtime_input=None):
        return self._effect("execute", runtime_input)

    def publish(self, payload=None, _options=None):
        return self._effect("publish", payload)

point_bindings = sdk_context.get("pointBindings") or {}
datapoints = sdk_context.get("datapoints") or {}
dp = {}
for alias, point_path in point_bindings.items() if isinstance(point_bindings, dict) else []:
    snapshot = datapoints.get(point_path)
    if isinstance(snapshot, dict):
        dp[alias] = DataPoint(snapshot)

class PointCollection:
    pass

points = PointCollection()
for alias, point in dp.items():
    setattr(points, alias, point)

class LoggerSDK:
    def log(self, *values):
        print(*values)

    info = log
    warn = log
    error = log

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
        self.points = points
        self.args = dict(input_data)
        self.trigger = dict(input_data.get("trigger") or {})
        self.runtime = dict(sdk_context.get("metadata") or {})
        self.logger = LoggerSDK()

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
    sys.stdout.write(json.dumps({"output": output, "sideEffects": side_effects, "logs": logs}, ensure_ascii=False))
except Exception:
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
