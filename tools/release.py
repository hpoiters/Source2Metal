"""Package final v3.4.1; publish only after source and consistency checks."""
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
VERSION = '3.4.1'
TAG = 'v' + VERSION
REPO = 'hpoiters/Source2Metal'
CORE = ROOT / 'source/Source2Metal_core'
OUT = ROOT / ('release_v' + VERSION)
LANGS = ('DE', 'EN', 'ES', 'FR', 'NL', 'RU', 'ZH')
DOCS = ('BUILDING.md', 'CODE_SIGNING_POLICY.md', 'LICENSE', 'PRIVACY.md',
        'README.md', 'RELEASE_NOTES.md', 'THIRD_PARTY_NOTICES.md', 'VALIDATION.md')
HTML_NAME = 'Readme-README-Прочтите-自述文件.html'
SOURCE_NAME = 'Source2Metal_v' + VERSION + '_SOURCE.zip'
PACKAGE_NAME = 'Source2Metal_v' + VERSION + '_RELEASE.zip'

def digest(data):
    return hashlib.sha256(data).hexdigest()

def verify():
    hashes = json.loads((ROOT / 'validation/release_hashes.json').read_text())
    for name, wanted in hashes.items():
        assert digest((ROOT / name).read_bytes()) == wanted, 'Identity mismatch: ' + name
    core = (ROOT / 'dist/Source2Metal_core.exe').read_bytes()
    launcher = (ROOT / ('dist/!Source2Metal_v' + VERSION + '.exe')).read_bytes()
    assert core in launcher, 'Launcher does not embed the exact core'
    config = ROOT / 'dist/Source2Metal.ini'
    previous = config.read_bytes() if config.exists() else None
    try:
        for lang in LANGS:
            config.write_text('Language=' + lang.lower() + '\n')
            info = subprocess.check_output([str(ROOT / 'dist/core-native'), '--build-info'], text=True)
            assert 'Source2Metal v3.4.1\n' in info
            assert 'Source2Metal_v3.4.1_SOURCE.zip' in info
            assert 'RC3' not in info and 'trial package' not in info
    finally:
        if previous is None:
            config.unlink(missing_ok=True)
        else:
            config.write_bytes(previous)
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
    for name in DOCS + ('tools/build_release.sh', 'validation/approved_console_hashes.json'):
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
    print('Verified package, source, embedded core and seven languages.')

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
    # GitHub's by-tag endpoint may omit drafts. Resolve the draft by its ID.
    matches = [r for r in api('releases?per_page=100') if r['tag_name'] == TAG]
    if matches:
        assert len(matches) == 1 and matches[0]['draft'], 'Existing public release; no overwrite'
        release = api('releases/' + str(matches[0]['id']))
        tag_sha = api('git/ref/tags/' + TAG)['object']['sha']
        if tag_sha != sha:
            comparison = api('compare/' + tag_sha + '...' + sha)
            assert comparison['status'] == 'ahead'
            assert {f['filename'] for f in comparison['files']} <= {'tools/release.py'}, 'Release inputs changed'
    else:
        assert all(t['name'] != TAG for t in api('tags?per_page=100')), 'Unrelated tag already exists'
        gh('api', '--method', 'POST', 'repos/' + REPO + '/git/refs',
           '-f', 'ref=refs/tags/' + TAG, '-f', 'sha=' + sha)
        tag_sha = sha
        gh('release', 'create', TAG, '--repo', REPO, '--verify-tag', '--draft',
           '--title', 'Source2Metal v3.4.1', '--notes-file', str(ROOT / 'RELEASE_NOTES.md'),
           *map(str, assets))
        release = next(r for r in api('releases?per_page=100') if r['tag_name'] == TAG)
    assert release['draft'] and {a['name'] for a in release['assets']} == names
    actual = {}
    for asset in release['assets']:
        data = subprocess.check_output(['gh', 'api', 'repos/' + REPO + '/releases/assets/' + str(asset['id']),
                                        '-H', 'Accept: application/octet-stream'])
        actual[asset['name']] = digest(data)
    assert actual == wanted, 'Uploaded release differs from verified local files'
    assert api('git/ref/tags/' + TAG)['object']['sha'] == tag_sha
    assert api('git/ref/heads/main')['object']['sha'] == sha
    gh('api', '--method', 'PATCH', 'repos/' + REPO + '/releases/' + str(release['id']),
       '-F', 'draft=false', '-F', 'prerelease=false', '-f', 'make_latest=true')
    release = api('releases/latest')
    assert release['tag_name'] == TAG and not release['prerelease'] and not release['draft']
    # Keep the previous candidate available as a clearly labelled prerelease.
    for old in api('releases?per_page=100'):
        if old['tag_name'] == 'v3.4.1-RC3':
            gh('api', '--method', 'PATCH', 'repos/' + REPO + '/releases/' + str(old['id']),
               '-F', 'prerelease=true', '-f', 'make_latest=false',
               '-f', 'name=Source2Metal v3.4.1-RC3 — superseded candidate')
    assert api('releases/latest')['tag_name'] == TAG
    print('Published and verified:', release['html_url'])

if __name__ == '__main__':
    {'package': package, 'publish': publish}[sys.argv[1]]()
