"""Exercise the v3.3.1 executable from the unpacked user ZIP on Windows."""
from pathlib import Path
import hashlib, struct, subprocess, sys, tempfile, zipfile

VERSION="3.3.1"

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
        # The first source starts with 100 unusable PGN results; a later game
        # and a second source must both reach the merged GAME RAW output.
        pgn_input=root/"PGN warning regression"; pgn_input.mkdir()
        (root/"Source2Metal.ini").write_text("Language=nl\n",encoding="utf-8")
        first=pgn_input/"01_unusable.pgn"
        first.write_text(''.join(f'[Event "line {i}"]\n[Result "*"]\n\n1. e4 e5 *\n\n' for i in range(100))
                         +'[Event "later"]\n[Result "1-0"]\n\n1. d4 d5 1-0\n',encoding="utf-8")
        (pgn_input/"02_next.pgn").write_text('[Event "next"]\n[Result "0-1"]\n\n1. c4 e5 0-1\n',encoding="utf-8")
        result=subprocess.run([str(exe),"-input",str(pgn_input),"-mode","raw","-min-ply","2","-workers","32","-no-pause"],cwd=root,capture_output=True,timeout=120)
        if result.returncode: raise RuntimeError(f"PGN warning: {result.stdout!r} {result.stderr!r}")
        output=(pgn_input/"!Source2Metal_Output")
        run,=output.iterdir()
        merged=run/"RAW"/"2 - Merged Sources - RAW PGNs"/"Source2Metal - Merged GAME Sources - RAW.pgn"
        text=merged.read_text(encoding="utf-8-sig")
        assert text.count('[Result "1-0"]')==1 and text.count('[Result "0-1"]')==1
        assert '100 of 100' in result.stdout.decode('utf-8',errors='replace') or '100 van 100' in result.stdout.decode('utf-8',errors='replace')
        print("100 unusable PGN games: later game and next source OK with 32 workers")
        assert hashlib.sha256(exe.read_bytes()).hexdigest()==original
        print("Broken BIN rejected; EXE unchanged; SHA-256:",original)

if __name__=="__main__": main()
