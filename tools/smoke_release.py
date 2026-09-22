"""Exercise the v3.3.0 executable from the unpacked user ZIP on Windows."""
from pathlib import Path
import hashlib, struct, subprocess, sys, tempfile, zipfile

VERSION="3.3.0"

def main():
    archive=Path(sys.argv[1]).resolve()
    with tempfile.TemporaryDirectory(prefix="Source2Metal Windows test ") as tmp:
        root=Path(tmp)/"Gebruiker Тест 测试"
        with zipfile.ZipFile(archive) as z: z.extractall(root)
        exe=root/f"!Source2Metal_v{VERSION}.exe"
        original=hashlib.sha256(exe.read_bytes()).hexdigest()
        record=struct.pack(">QHHI",0x463B96181691FC9C,0x031C,100,0)
        for lang in ("en","de","nl","fr","es","ru","zh"):
            (root/"Source2Metal.ini").write_text("Language="+lang+"\n",encoding="utf-8")
            inputs=root/("Input "+lang); inputs.mkdir()
            for folder in ("a","b"):
                target=inputs/folder/"same.BIN"; target.parent.mkdir(); target.write_bytes(record)
            result=subprocess.run([str(exe),"-input",str(inputs),"-mode","all","-min-ply","24","-min-elo","2500","-workers","1","-no-pause"],cwd=root,capture_output=True,timeout=120)
            if result.returncode: raise RuntimeError(f"{lang}: {result.stdout!r} {result.stderr!r}")
            run,=(inputs/"!Source2Metal_Output").iterdir()
            separate=list((run/"RAW"/"1 - Separate Sources - RAW PGNs"/"BOOK Sources").glob("*.pgn"))
            assert len(separate)==2
            merged=run/"RAW"/"2 - Merged Sources - RAW PGNs"/"Source2Metal - Merged BOOK Sources - RAW.pgn"
            for p in separate+[merged]:
                text=p.read_text(encoding="utf-8-sig"); assert text.count('[Result "*"]')==1 and "1. e4 *" in text
            report=(run/"REPORTS"/"Source2Metal_Report.txt").read_text(encoding="utf-8-sig")
            assert f"Source2Metal v{VERSION}" in report
            print(lang+": packaged EXE OK")
        bad=root/"Broken input"; bad.mkdir(); (bad/"broken.bin").write_bytes(b"broken")
        result=subprocess.run([str(exe),"-input",str(bad),"-mode","raw","-no-pause"],cwd=root,capture_output=True,timeout=120)
        assert result.returncode!=0
        assert hashlib.sha256(exe.read_bytes()).hexdigest()==original
        print("Broken BIN rejected; EXE unchanged; SHA-256:",original)

if __name__=="__main__": main()
