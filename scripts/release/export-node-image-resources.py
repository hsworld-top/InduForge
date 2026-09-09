#!/usr/bin/env python3
"""将本地 docker save 归档拆成中心对象库资源；不访问镜像仓库或网络。"""
import argparse
import gzip
import hashlib
import io
import json
from pathlib import Path
import tarfile
import tempfile


def export(source, destination, architecture, append=False):
    destination = Path(destination)
    archives = destination / "archives"
    archives.mkdir(parents=True, exist_ok=True)
    artifacts = []
    with tarfile.open(source, "r:*") as bundle:
        manifest = json.load(bundle.extractfile("manifest.json"))
        for entry in manifest:
            references = entry.get("RepoTags") or []
            if not references:
                continue
            config = bundle.extractfile(entry["Config"]).read()
            actual_arch = json.loads(config).get("architecture")
            if actual_arch != architecture:
                raise ValueError(f"镜像架构不一致: {references}: {actual_arch}")
            config_digest = "sha256:" + hashlib.sha256(config).hexdigest()
            # 不解压到文件系统，避免归档路径穿越；只复制 manifest 引用的普通文件。
            with tempfile.NamedTemporaryFile(dir=archives, suffix=".partial", delete=False) as temporary:
                temporary_path = Path(temporary.name)
            try:
                with temporary_path.open("wb") as raw, gzip.GzipFile(filename="", mode="wb", fileobj=raw, mtime=0, compresslevel=3) as compressed, tarfile.open(fileobj=compressed, mode="w") as output:
                    for name in dict.fromkeys([entry["Config"], *entry["Layers"]]):
                        member = bundle.getmember(name)
                        if not member.isfile() or name.startswith("/") or ".." in Path(name).parts:
                            raise ValueError(f"非法镜像归档成员: {name}")
                        output.addfile(member, bundle.extractfile(member))
                    data = json.dumps([entry], separators=(",", ":")).encode()
                    member = tarfile.TarInfo("manifest.json")
                    member.size = len(data)
                    output.addfile(member, io.BytesIO(data))
                with temporary_path.open("rb") as archive_file:
                    hasher = hashlib.sha256()
                    for chunk in iter(lambda: archive_file.read(1024 * 1024), b""):
                        hasher.update(chunk)
                    digest = hasher.hexdigest()
                target = archives / (digest + ".tar.gz")
                size = temporary_path.stat().st_size
                temporary_path.replace(target)
                artifacts.append({
                    "architecture": architecture,
                    "sha256": digest,
                    "size": size,
                    "archive": "archives/" + target.name,
                    "images": [{"reference": ref, "configDigest": config_digest} for ref in references],
                })
            finally:
                temporary_path.unlink(missing_ok=True)
    if not artifacts:
        raise ValueError("归档没有带标签的镜像")
    manifest_path = destination / "manifest.json"
    previous = json.loads(manifest_path.read_text()) if manifest_path.exists() else {"schemaVersion": 1, "artifacts": []}
    keys = {(a["architecture"], im["reference"]) for a in artifacts for im in a["images"]}
    previous["artifacts"] = [a for a in previous["artifacts"] if (a["architecture"] != architecture if not append else not any((a["architecture"], im["reference"]) in keys for im in a["images"]))] + artifacts
    temp = manifest_path.with_suffix(".partial")
    temp.write_text(json.dumps(previous, ensure_ascii=False, indent=2) + "\n")
    temp.replace(manifest_path)
    return artifacts


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", required=True, help="现有 docker save tar 或 tar.gz")
    parser.add_argument("--output", required=True)
    parser.add_argument("--architecture", choices=["amd64", "arm64"], required=True)
    parser.add_argument("--append", action="store_true", help="向现有目录补充镜像")
    args = parser.parse_args()
    items = export(args.archive, args.output, args.architecture, args.append)
    print(f"已生成 {len(items)} 项 {args.architecture} 离线镜像资源")
