#!/usr/bin/env python3
"""从中心安装资源目录导入对象库；凭据只传给导入进程，不打印或落盘。"""
import base64
import json
import os
from pathlib import Path
import subprocess
import sys

root = Path(__file__).resolve().parent
directory = Path(sys.argv[1]) if len(sys.argv) > 1 else root / "resources"
binary = Path(os.environ.get("IF_CENTER_IMAGE_IMPORT_BINARY", str(root / "bin/image-import")))
kubectl = [os.environ.get("IF_CENTER_K3S_BIN", "/usr/local/bin/k3s"), "kubectl", "-n", "induforge-system"]
if os.environ.get("IF_CENTER_KUBECONFIG"):
    kubectl += ["--kubeconfig", os.environ["IF_CENTER_KUBECONFIG"]]
secret = json.loads(subprocess.check_output(kubectl + ["get", "secret", "center-env", "-o", "json"]))
service = json.loads(subprocess.check_output(kubectl + ["get", "service", "center-object-store", "-o", "json"]))
env = dict(os.environ)
for key, value in secret["data"].items():
    if key.startswith("IF_OBJECT_STORE_"):
        env[key] = base64.b64decode(value).decode()
env["IF_OBJECT_STORE_ENDPOINT"] = service["spec"]["clusterIP"]
env["IF_OBJECT_STORE_PORT"] = str(next(p["port"] for p in service["spec"]["ports"] if p["name"] == "s3"))
env["IF_OBJECT_STORE_USE_SSL"] = "false"
subprocess.run([str(binary), "--dir", str(directory)], env=env, check=True)
