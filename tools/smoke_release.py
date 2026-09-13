"""Exercise the executable from the unpacked user ZIP on Windows."""
from pathlib import Path
import hashlib
import struct
import subprocess
import sys
import tempfile
import zipfile


def main():
    archive = Path(sys.argv[1]).resolve()
    with tempfile.TemporaryDirectory(prefix="Source2Metal Windows test ") as tmp:
        root = Path(tmp) / "Gebruiker Тест 测试"
        with zipfile.ZipFile(archive) as z:
            z.extractall(root)
        exe, = root.glob("!Source2Metal_v*.exe")
        original = hashlib.sha256(exe.read_bytes()).hexdigest()
        record = struct.pack(">QHHI", 0x463B96181691FC9C, 0x031C, 100, 0)
        for lang in ("en", "de", "nl", "fr", "es", "ru", "zh"):
            (root / "Source2Metal.ini").write_text("Language=" + lang + "\n", encoding="utf-8")
            inputs = root / ("Input " + lang)
            inputs.mkdir()
            for folder in ("a", "b"):
                target = inputs / folder / "same.BIN"
                target.parent.mkdir()
                target.write_bytes(record)
            result = subprocess.run(
                [str(exe), "-input", str(inputs), "-mode", "all", "-min-ply", "24",
                 "-min-elo", "2500", "-workers", "1", "-no-pause"],
                cwd=root, capture_output=True, timeout=120)
            if result.returncode:
                raise RuntimeError(f"{lang}: EXE failed: {result.stdout!r} {result.stderr!r}")
            run, = (inputs / "!Source2Metal_Output").iterdir()
            raw = run / "RAW"
            separate = list((raw / "1 - Separate Sources - RAW PGNs" / "BOOK Sources").glob("*.pgn"))
            assert len(separate) == 2, (lang, separate)
            merged = raw / "2 - Merged Sources - RAW PGNs" / "Source2Metal - Merged BOOK Sources - RAW.pgn"
            for p in separate + [merged]:
                text = p.read_text(encoding="utf-8-sig")
                assert text.count('[Result "*"]') == 1 and "1. e4 *" in text, (lang, p, text)
            for p in (inputs / "a" / "same.BIN", inputs / "b" / "same.BIN"):
                assert p.read_bytes() == record, "Input changed"
            status = (run / "REPORTS" / "STATUS.txt").read_text(encoding="utf-8-sig")
            assert "BIN2PGN 0.1.3" in status and "%!" not in status, (lang, status)
            print(f"{lang}: packaged EXE, Unicode/spaces, duplicate BIN, default GAME filters: OK")
        bad = root / "Broken input"
        bad.mkdir()
        (bad / "broken.bin").write_bytes(b"broken")
        result = subprocess.run([str(exe), "-input", str(bad), "-mode", "raw", "-no-pause"],
                                cwd=root, capture_output=True, timeout=120)
        assert result.returncode != 0, "Broken BIN returned success"
        (root / "Source2Metal.ini").unlink()
        result = subprocess.run([str(exe)], input=b"3\n0\n", cwd=root,
                                capture_output=True, timeout=30)
        assert result.returncode == 0, "Interactive language selection/exit failed"
        assert "Language=nl" in (root / "Source2Metal.ini").read_text(), "Language not saved"
        assert hashlib.sha256(exe.read_bytes()).hexdigest() == original, "Executable changed"
        print("Broken BIN rejected; interactive Dutch selection remembered; EXE unchanged: OK")
        print("Tested EXE SHA-256:", original)


if __name__ == "__main__":
    main()
