from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import sys
import tempfile
import unittest


PROFILE = Path(__file__).resolve().parents[1] / "profiles/menu_supersession.py"
spec = importlib.util.spec_from_file_location("menu_supersession_profile_test", PROFILE)
if spec is None or spec.loader is None:
    raise RuntimeError("cannot load menu supersession profile")
profile = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = profile
spec.loader.exec_module(profile)


def artifact(cache: Path, module: str, version: str, suffix: str, content: str) -> None:
    path = cache.joinpath(*profile.proxy_escape(module).split("/"), "@v")
    path.mkdir(parents=True, exist_ok=True)
    (path / f"{profile.proxy_escape(version)}{suffix}").write_text(content, encoding="ascii")


class MenuSupersessionProfileTests(unittest.TestCase):
    def test_missing_unrelated_old_info_does_not_block_required_artifacts(self) -> None:
        with tempfile.TemporaryDirectory(prefix="go-source-artifact-test-") as directory:
            cache = Path(directory).resolve()
            artifact(cache, "google.golang.org/protobuf", "v1.26.0-rc.1", ".mod", "module google.golang.org/protobuf\n")
            artifact(cache, "google.golang.org/protobuf", "v1.36.10", ".mod", "module google.golang.org/protobuf\n")
            artifact(cache, "google.golang.org/protobuf", "v1.36.10", ".zip", "fixture")
            artifact(cache, "google.golang.org/protobuf", "v1.36.10", ".ziphash", "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n")
            profile.validate_source_artifacts(
                cache,
                {"google.golang.org/protobuf": "v1.36.10"},
                {"google.golang.org/protobuf": "v1.36.10"},
            )
            self.assertFalse(
                (cache / "google.golang.org/protobuf/@v/v1.26.0-rc.1.info").exists()
            )

    def test_missing_required_mod_is_unverified(self) -> None:
        with tempfile.TemporaryDirectory(prefix="go-source-artifact-test-") as directory:
            with self.assertRaises(SystemExit):
                profile.validate_source_artifacts(
                    Path(directory).resolve(),
                    {"example.com/required": "v1.2.3"},
                    {"example.com/required": "v1.2.3"},
                )

    def test_ambient_proxy_credentials_and_goenv_are_not_inherited(self) -> None:
        poison = {
            "HTTP_PROXY": "http://attacker.invalid",
            "HTTPS_PROXY": "http://attacker.invalid",
            "ALL_PROXY": "socks5://attacker.invalid",
            "NO_PROXY": "attacker.invalid",
            "GOPRIVATE": "attacker.invalid",
            "GONOPROXY": "attacker.invalid",
            "GOVCS": "*:all",
            "GOENV": "/tmp/attacker-goenv",
            "NETRC": "/tmp/attacker-netrc",
        }
        previous = {key: os.environ.get(key) for key in poison}
        try:
            os.environ.update(poison)
            with tempfile.TemporaryDirectory(prefix="go-controlled-env-test-") as directory:
                root = Path(directory).resolve()
                cache = root / "proxy"
                work = root / "work"
                go_binary = root / "go-bin"
                cache.mkdir()
                work.mkdir()
                go_binary.write_text("fixture", encoding="ascii")
                populate, offline = profile.controlled_go_environment(
                    work, str(go_binary), cache
                )
                for key in poison:
                    if key == "GOENV":
                        self.assertEqual(populate[key], "off")
                        self.assertEqual(offline[key], "off")
                    else:
                        self.assertNotIn(key, populate)
                        self.assertNotIn(key, offline)
                self.assertEqual(populate["GOPROXY"], cache.as_uri())
                self.assertEqual(offline["GOPROXY"], "off")
        finally:
            for key, value in previous.items():
                if value is None:
                    os.environ.pop(key, None)
                else:
                    os.environ[key] = value


if __name__ == "__main__":
    unittest.main()
