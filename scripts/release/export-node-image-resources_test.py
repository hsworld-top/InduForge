import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("exporter", Path(__file__).with_name("export-node-image-resources.py"))
exporter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(exporter)

class ResourceExportTest(unittest.TestCase):
    def test_archive_is_self_contained_and_rejects_wrong_architecture(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source = root / "images.tar"
            with tarfile.open(source, "w") as archive:
                files = {
                    "config.json": json.dumps({"architecture": "arm64"}).encode(),
                    "layer.tar": b"layer",
                    "manifest.json": json.dumps([{"Config": "config.json", "Layers": ["layer.tar"], "RepoTags": ["example:test"]}]).encode(),
                }
                for name, data in files.items():
                    item = tarfile.TarInfo(name)
                    item.size = len(data)
                    archive.addfile(item, io.BytesIO(data))
            items = exporter.export(source, root / "out", "arm64")
            self.assertEqual(len(items), 1)
            item = items[0]
            with tarfile.open(root / "out" / item["archive"]) as archive:
                self.assertEqual(archive.extractfile("layer.tar").read(), b"layer")
                self.assertEqual(len(json.load(archive.extractfile("manifest.json"))), 1)
            with self.assertRaises(ValueError):
                exporter.export(source, root / "out", "amd64")
            self.assertEqual(len(json.loads((root / "out/manifest.json").read_text())["artifacts"]), 1)

if __name__ == "__main__":
    unittest.main()
