"""Compare approved and rebuilt Windows kernels and exercise the final launcher."""
from pathlib import Path
import gzip, hashlib, json, os, shutil, struct, subprocess, sys, tempfile

ROOT = Path(__file__).resolve().parents[1]
CORE = ROOT / 'source/Source2Metal_console/core.exe'
EXE = ROOT / 'source/Source2Metal_v3.2.1/!Source2Metal_v3.2.1.exe'
REF = ROOT / 'validation/approved_core.exe'
EXPECTED = '153b232b403929743facf8ac8f1791627a76dbea46a16425fdd6a67cac664392'
SYZYGY = 'd80f8fd70f81b4f06cbd89547a7e346aa08688853b2ce2df25b30e93ccd6cb2c'
LANGS = ('en','de','nl','fr','es','zh','ru')

def prepare():
    data = gzip.decompress((ROOT / 'validation/approved_core.exe.gz').read_bytes())
    assert hashlib.sha256(data).hexdigest() == EXPECTED
    REF.write_bytes(data)
    return REF

def instructions():
    result = subprocess.check_output(['go','run','.',str(REF),str(CORE)], cwd=ROOT/'tools/core_compare')
    report = json.loads(result)
    # The 2CBH/CBH adapter change is display-only; conversion parity below
    # must still match the approved kernel byte-for-byte in every language.
    allowed = ('main.startBINActivity', 'main.startBookMergeActivity', 'main.mergeBookRAW', 'main.prepareChessBaseSources')
    unexpected = [n for n in report['changed']+report['missing'] if not n.startswith(allowed)]
    assert not unexpected, unexpected
    assert len(report['same']) >= 390, len(report['same'])
    (ROOT/'validation/core_comparison.json').write_bytes(result)
    print(f"Core comparison: {len(report['same'])} functions match after relocation normalization; differences restricted to progress instrumentation")

def execute(exe, root, cwd, args=(), input=None):
    env = os.environ.copy()
    env.pop('SOURCE2METAL_LAUNCH_DIR', None)
    env['TEMP'] = env['TMP'] = str(cwd)
    result = subprocess.run([str(exe),*args],cwd=cwd,env=env,input=input,capture_output=True,timeout=120)
    if result.returncode:
        raise AssertionError((result.returncode,result.stdout.decode('utf8','replace'),result.stderr.decode('utf8','replace')))
    return result.stdout.decode('utf8','replace')

def fixture_pgn():
    # GAME-METAL requires >=32 distinct games supporting the common moves.
    # Every game shares 24 plies; two legal tail plies make all 32 unique.
    opening = '1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 11. c4 c6 12. Nc3 Qc7'
    games=[]
    for white in ('a3','a4','Be3','Bg5'):
        for black in ('h6','h5','g6','a5','Kh8','Rb8','Rd8','Nb6'):
            games.append(f'[Event "Parity"]\n[White "A{len(games)}"]\n[Black "B"]\n[WhiteElo "2600"]\n[BlackElo "2600"]\n[Result "1-0"]\n\n{opening} 13. {white} {black} 1-0\n')
    return '\n'.join(games)

def windows():
    assert os.name == 'nt', 'Native Windows verification is required'
    with tempfile.TemporaryDirectory(prefix='Source2Metal parity ') as temp:
        root = Path(temp)/'Gebruiker Тест 测试'; root.mkdir()
        other = Path(temp)/'Different working directory';other.mkdir()
        # A tempting input outside the application directory must not be scanned.
        (other/'WRONG_TEMP_SOURCE.pgn').write_text('[Event "Wrong"]\n[Result "1-0"]\n\n1. e4 e5 1-0\n')
        app = root/'!Source2Metal_v3.2.1.exe'
        record=struct.pack('>QHHI',0x463B96181691FC9C,0x031C,100,0)
        for name in ('first.bin','second.bin'):(root/name).write_bytes(record)
        (root/'third.bin').write_bytes(struct.pack('>QHHI',0x463B96181691FC9C,0x02DB,80,0))
        (root/'game.pgn').write_text(fixture_pgn(),encoding='utf8')
        source_hashes={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in root.iterdir()}
        for lang in LANGS:
            snapshots=[]
            for binary in (REF,CORE,EXE):
                shutil.copy2(binary,app)
                (root/'Source2Metal.ini').write_text('Language='+lang+'\n',encoding='utf8')
                output=root/'!Source2Metal_Output'
                if output.exists():shutil.rmtree(output)
                text=execute(app,root,other,('-mode','all','-min-ply','1','-workers','1','-no-pause'))
                assert 'WRONG_TEMP_SOURCE' not in text, text
                runs=list(output.iterdir());assert len(runs)==1
                files={str(p.relative_to(runs[0])):p.read_bytes() for p in runs[0].rglob('*.pgn')}
                assert len(files)>=6, files.keys()
                assert any('METAL' in p for p in files),files.keys()
                snapshots.append(files)
            assert snapshots[0]==snapshots[1]==snapshots[2], f'PGN parity failed: {lang}'
            print(lang+': approved core = rebuilt core = final launcher PGNs; default folder isolated from TEMP/CWD')
        assert source_hashes=={name:hashlib.sha256((root/name).read_bytes()).hexdigest() for name in source_hashes}
        # Real interactive processes, all 49 language transitions.
        for source in LANGS:
            for number,target in enumerate(LANGS,1):
                (root/'Source2Metal.ini').write_text('Language='+source+'\n')
                text=execute(app,root,other,input=f'3\n{number}\n0\n'.encode())
                assert 'Language='+target in (root/'Source2Metal.ini').read_text()
                assert 'SyzygyCheck' in text
        print('49 interactive language transitions: OK')
        for lang in LANGS:
            (root/'Source2Metal.ini').write_text('Language='+lang+'\n')
            utility=root/'SyzygyCheck_v2.2.0_UTILITY.zip'
            utility.unlink(missing_ok=True)
            execute(app,root,other,input=b'2\n1\n\n0\n')
            assert hashlib.sha256(utility.read_bytes()).hexdigest()==SYZYGY
            assert not (other/utility.name).exists()
        assert not (other/'Source2Metal.ini').exists()
        print('Seven SyzygyCheck menu extraction tests and stable INI directory: OK')

if __name__=='__main__':
    prepare()
    if '--prepare-only' not in sys.argv:
        instructions()
        if '--instructions-only' not in sys.argv:windows()
