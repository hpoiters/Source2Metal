"""Package the unchanged Windows-tested RC3; publish only after identity checks."""
import hashlib
import html
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
VERSION = '3.4.1-RC3'
TAG = 'v' + VERSION
REPO = 'hpoiters/Source2Metal'
CORE = ROOT / 'source/Source2Metal_core'
OUT = ROOT / ('release_v' + VERSION)
LANGS = ('DE', 'EN', 'ES', 'FR', 'NL', 'RU', 'ZH')
DOCS = ('BUILDING.md', 'CODE_SIGNING_POLICY.md', 'LICENSE', 'PRIVACY.md',
        'README.md', 'RELEASE_NOTES_RC3.md', 'THIRD_PARTY_NOTICES.md', 'VALIDATION_RC3.md')
HTML_NAME = 'Readme-README-Прочтите-自述文件.html'
SOURCE_NAME = 'Source2Metal_v' + VERSION + '_SOURCE.zip'
PACKAGE_NAME = 'Source2Metal_v' + VERSION + '_RELEASE.zip'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def verify():
    hashes = json.loads((ROOT / 'validation/rc3_release_hashes.json').read_text())
    for name, wanted in hashes.items():
        assert digest((ROOT / name).read_bytes()) == wanted, 'Identity mismatch: ' + name
    core = (ROOT / 'dist/Source2Metal_core.exe').read_bytes()
    launcher = (ROOT / ('dist/!Source2Metal_v' + VERSION + '.exe')).read_bytes()
    assert core in launcher, 'Launcher does not embed the exact core'
    page = (CORE / HTML_NAME).read_text()
    sections = dict(re.findall(r'<section id="MANUAL_(\w+)"><pre>(.*?)</pre></section>', page, re.S))
    assert set(sections) == set(LANGS)
    for lang in LANGS:
        general = (CORE / ('MANUAL_' + lang + '.txt')).read_text()
        overlay = (CORE / 'OVERLAY-PGN' / ('MANUAL_' + lang + '.txt')).read_text()
        assert html.unescape(sections[lang]) == general, 'HTML differs: ' + lang
        for text in (general, overlay):
            for token in (TAG, 'VoorbeeldOverlay.pgn', '.overlay.pgn', 'RAW + METAL', '100'):
                assert token in text, (lang, token)
        assert 'covered' not in general
        assert 'Latest' not in general
        for token in ('Read 6', 'Accepted 6', 'Deduplicated 1', 'RAW 5', 'Rejected 0',
                      'Final Metal contribution 5', 'Final Metal verified records 5'):
            assert token in overlay, (lang, token)
    assert (CORE / 'OVERLAY-PGN/VoorbeeldOverlay.pgn').read_text().count('[Event ') == 6

def archive(files):
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for name, data in sorted(files.items()):
            info = zipfile.ZipInfo(name, (2026, 9, 29, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            z.writestr(info, data, compresslevel=9)
    return buf.getvalue()

def package():
    verify()
    source = {}
    for directory in ('source/Source2Metal_core', 'source/Source2Metal_console'):
        for p in (ROOT / directory).rglob('*'):
            if p.is_file() and p.suffix != '.exe':
                source[p.relative_to(ROOT).as_posix()] = p.read_bytes()
    for name in DOCS + ('tools/build_rc3.sh', 'validation/approved_console_hashes.json'):
        source[name] = (ROOT / name).read_bytes()
    files = {SOURCE_NAME: archive(source)}
    for name in DOCS:
        files['Docs-DOCUMENTATION-Документация-文档/' + name] = (ROOT / name).read_bytes()
    for lang in LANGS:
        name = 'MANUAL_' + lang + '.txt'
        files['Docs-DOCUMENTATION-Документация-文档/' + name] = (CORE / name).read_bytes()
    for p in (CORE / 'OVERLAY-PGN').iterdir():
        if p.is_file():
            files['OVERLAY-PGN/' + p.name] = p.read_bytes()
    files[HTML_NAME] = (CORE / HTML_NAME).read_bytes()
    files['!Source2Metal_v' + VERSION + '.exe'] = (ROOT / ('dist/!Source2Metal_v' + VERSION + '.exe')).read_bytes()
    files['Development/Source2Metal_core.exe'] = (ROOT / 'dist/Source2Metal_core.exe').read_bytes()
    files['SHA256_CONTENTS.txt'] = ''.join(digest(b) + '  ' + n + '\n' for n, b in sorted(files.items())).encode()
    packed = archive(files)
    with zipfile.ZipFile(io.BytesIO(packed)) as z:
        assert z.testzip() is None
        for name, data in files.items():
            assert z.read(name) == data
    OUT.mkdir(exist_ok=True)
    (OUT / PACKAGE_NAME).write_bytes(packed)
    (OUT / SOURCE_NAME).write_bytes(files[SOURCE_NAME])
    sums = OUT / ('SHA256_Source2Metal_v' + VERSION + '.txt')
    sums.write_text(''.join(digest((OUT / n).read_bytes()) + '  ' + n + '\n' for n in (PACKAGE_NAME, SOURCE_NAME)))
    print('Verified package, source, binary identity and seven languages.')

def gh(*args):
    return subprocess.check_output(['gh', *args], text=True)

def api(path):
    return json.loads(gh('api', 'repos/' + REPO + '/' + path))

def publish():
    verify()
    assert os.environ['GITHUB_REF'] == 'refs/heads/main'
    sha = os.environ['GITHUB_SHA']
    assert api('git/ref/heads/main')['object']['sha'] == sha, 'Main changed during verification'
    names = {PACKAGE_NAME, SOURCE_NAME, 'SHA256_Source2Metal_v' + VERSION + '.txt'}
    assets = sorted(OUT.iterdir())
    assert {p.name for p in assets} == names
    wanted = {p.name: digest(p.read_bytes()) for p in assets}
    assert all(t['name'] != TAG for t in api('tags?per_page=100')), 'Tag already exists; no overwrite'
    gh('api', '--method', 'POST', 'repos/' + REPO + '/git/refs',
       '-f', 'ref=refs/tags/' + TAG, '-f', 'sha=' + sha)
    gh('release', 'create', TAG, '--repo', REPO, '--verify-tag', '--draft',
       '--title', 'Source2Metal v3.4.1-RC3', '--notes-file', str(ROOT / 'RELEASE_NOTES_RC3.md'),
       *map(str, assets))
    release = api('releases/tags/' + TAG)
    assert release['draft'] and {a['name'] for a in release['assets']} == names
    with tempfile.TemporaryDirectory() as tmp:
        gh('release', 'download', TAG, '--repo', REPO, '--dir', tmp)
        actual = {p.name: digest(p.read_bytes()) for p in Path(tmp).iterdir()}
        assert actual == wanted, 'Uploaded release differs from verified local files'
    assert api('git/ref/tags/' + TAG)['object']['sha'] == sha
    assert api('git/ref/heads/main')['object']['sha'] == sha
    gh('release', 'edit', TAG, '--repo', REPO, '--draft=false', '--prerelease=false', '--latest')
    release = api('releases/latest')
    assert release['tag_name'] == TAG and not release['prerelease'] and not release['draft']
    print('Published and verified:', release['html_url'])

if __name__ == '__main__':
    {'package': package, 'publish': publish}[sys.argv[1]]()
