"""Build and verify the Source2Metal v3.3.0 user package."""
from pathlib import Path
from html.parser import HTMLParser
from urllib.parse import unquote, urlsplit
import hashlib
import zipfile

ROOT = Path(__file__).resolve().parents[1]
VERSION = "3.3.0"
SOURCE = ROOT / "source" / f"Source2Metal_v{VERSION}"
OUTPUT = ROOT / f"release_v{VERSION}"
INTRO = "Readme-README-Прочтите-自述文件.html"
DOCS = "Docs-DOCUMENTATION-Документация-文档"
EXE = f"!Source2Metal_v{VERSION}.exe"

class Links(HTMLParser):
    def __init__(self):
        super().__init__(); self.links=[]; self.ids=set()
    def handle_starttag(self, tag, attrs):
        a=dict(attrs)
        if "id" in a: self.ids.add(a["id"])
        if tag=="a" and "href" in a: self.links.append(a["href"])

def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()

def main():
    OUTPUT.mkdir(exist_ok=True)
    files={EXE:SOURCE/EXE, INTRO:SOURCE/INTRO}
    for lang in ("EN","DE","NL","FR","ES","RU","ZH"):
        name=f"MANUAL_{lang}.txt"; p=SOURCE/name
        text=p.read_text(encoding="utf-8-sig")
        for marker in (VERSION,"CTG METAL","BOOK","METAL","2.2.0"):
            if marker not in text: raise ValueError(f"{name}: missing {marker}")
        for obsolete in ("3.3.0-TEST","TEST package","TEST-pakket","TEST-Paket","Version TEST","Версия TEST","TEST 版本"):
            if obsolete in text: raise ValueError(f"{name}: obsolete marker {obsolete}")
        files[f"{DOCS}/{name}"]=p
    for name in (f"RELEASE_NOTES_v{VERSION}.txt","VALIDATION_REPORT_NL.txt","LICENSE_GPL-3.0.txt","THIRD_PARTY_NOTICES.txt"):
        files[f"{DOCS}/{name}"]=SOURCE/name
    for name in ("PRIVACY.md","THIRD_PARTY_NOTICES.md","CODE_SIGNING_POLICY.md","BUILDING.md"):
        files[f"{DOCS}/{name}"]=ROOT/name
    for arcname,p in files.items():
        if not p.is_file() or p.stat().st_size==0: raise ValueError(f"Missing/empty: {arcname}")
    html=(SOURCE/INTRO).read_text(encoding="utf-8-sig")
    if "3.3.0-TEST" in html: raise ValueError("HTML still identifies a test build")
    parser=Links(); parser.feed(html)
    for href in parser.links:
        u=urlsplit(href)
        if u.scheme or u.netloc: continue
        path=unquote(u.path).removeprefix("./").replace("\\","/")
        if path and path not in files and path.rstrip("/")!=DOCS: raise ValueError(f"Broken local link: {href}")
        if not path and u.fragment and unquote(u.fragment) not in parser.ids: raise ValueError(f"Broken anchor: {href}")
    for lang in ("NL","EN","DE","FR","ES","RU","ZH"):
        if lang not in parser.ids: raise ValueError(f"Missing intro language: {lang}")
    archive=OUTPUT/f"Source2Metal_v{VERSION}_RELEASE.zip"
    with zipfile.ZipFile(archive,"w",zipfile.ZIP_DEFLATED,compresslevel=9) as z:
        for name,p in sorted(files.items()):
            info=zipfile.ZipInfo(name,(2026,9,22,0,0,0)); info.compress_type=zipfile.ZIP_DEFLATED; info.external_attr=0o100644<<16
            z.writestr(info,p.read_bytes())
    with zipfile.ZipFile(archive) as z:
        if z.testzip() is not None: raise ValueError("ZIP integrity error")
        roots={p.split("/")[0] for p in z.namelist()}
        if roots!={EXE,INTRO,DOCS}: raise ValueError(f"Unexpected roots: {roots}")
        if any(p.endswith((".go",".bin",".ctg")) for p in z.namelist()): raise ValueError("Source/book data in user ZIP")
    sha=OUTPUT/f"SHA256_Source2Metal_v{VERSION}.txt"
    sha.write_text(f"{digest(SOURCE/EXE)}  {EXE}\n{digest(archive)}  {archive.name}\n",encoding="utf-8")
    print(f"Release ZIP OK: {len(files)} files; seven manuals and HTML links checked")
    print(sha.read_text(encoding="utf-8"))

if __name__=="__main__": main()
