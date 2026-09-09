"""仅临时目录与模拟 systemctl；不触碰宿主业务服务或系统文件。"""
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


class UninstallTest(unittest.TestCase):
    def test_already_uninstalled_backs_up_without_starting_service(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            package = root / 'package'
            package.mkdir()
            script = package / 'uninstall.sh'
            script.write_text(Path(__file__).with_name('uninstall-linux.sh').read_text())
            mock = root / 'mock'
            mock.mkdir()
            for name, body in {
                'id': 'echo 0',
                'systemctl': 'echo "$*" >> "$TEST_CALLS"; case "$1" in show) echo not-found;; daemon-reload) :;; *) exit 99;; esac',
                'rm': 'echo "rm $*" >> "$TEST_CALLS"',
            }.items():
                target = mock / name
                target.write_text('#!/bin/sh\n' + body + '\n')
                target.chmod(0o755)
            config = root / 'config'
            config.mkdir()
            (config / 'config.yaml').write_text('retained configuration')
            for relative in ['runtime/images/cache.bin', 'runtime/deployments/project.bin', 'node-agent/logs/node.log']:
                target = root / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text('must not be backed up')
            (root/'runtime/ops-agent-identity.json').write_text('{}')
            env = dict(os.environ, PATH=str(mock)+':'+os.environ['PATH'],
                       PREFIX=str(root/'node-agent'), CONFIG_DIR=str(config),
                       RUNTIME_DATA_DIR=str(root/'runtime'), TEST_CALLS=str(root/'calls'))
            result = subprocess.run(['bash', str(script), '--yes', '--keep-data', '--backup-dir', str(root/'backup')], env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('节点服务已卸载', result.stdout)
            self.assertNotIn('start ', (root/'calls').read_text())
            archives = list((root/'backup').glob('*.tar.gz'))
            self.assertEqual(len(archives), 1)
            with tarfile.open(archives[0]) as archive:
                content = archive.extractfile(str(config/'config.yaml').lstrip('/')).read()
                self.assertEqual(content, b'retained configuration')
                names = archive.getnames()
                self.assertIn(str(root/'runtime/ops-agent-identity.json').lstrip('/'), names)
                self.assertFalse(any(name.endswith(('.bin', '.log')) for name in names))
            self.assertEqual(archives[0].stat().st_mode & 0o777, 0o600)
            # 备份目的地在源目录内时，必须在任何服务操作前失败。
            (root/'calls').unlink()
            result = subprocess.run(['bash', str(script), '--yes', '--backup-dir', str(config/'backup')], env=env, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse((root/'calls').exists())

if __name__ == '__main__':
    unittest.main()
