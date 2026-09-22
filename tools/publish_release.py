"""Publish immutable Source2Metal v3.3.0 as Latest after CI succeeds."""
import hashlib, json, os, subprocess
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
REPO="hpoiters/Source2Metal"; VERSION="3.3.0"; TAG="v3.3.0"

def api(path):
    return json.loads(subprocess.check_output(["gh","api",f"repos/{REPO}/{path}"],text=True,encoding="utf-8"))

def main():
    assert os.environ["GITHUB_REF"]=="refs/heads/main"
    sha=os.environ["GITHUB_SHA"]
    assert api("git/ref/heads/main")["object"]["sha"]==sha,"Main moved during build"
    assert all(t["name"]!=TAG for t in api("tags?per_page=100")),"v3.3.0 already exists; refusing replacement"
    folder=ROOT/f"release_v{VERSION}"; files=sorted(folder.iterdir())
    expected={f"SHA256_Source2Metal_v{VERSION}.txt",f"Source2Metal_v{VERSION}_RELEASE.zip"}
    assert {p.name for p in files}==expected,"Unexpected release assets"
    notes=ROOT/"source"/f"Source2Metal_v{VERSION}"/f"RELEASE_NOTES_v{VERSION}.txt"
    subprocess.run(["gh","release","create",TAG,*map(str,files),"--repo",REPO,"--target",sha,
                    "--title","Source2Metal v3.3.0 — complete CTG graph extraction",
                    "--notes-file",str(notes),"--latest"],check=True)
    release=api("releases/tags/"+TAG)
    actual={a["name"]:a["digest"].removeprefix("sha256:") for a in release["assets"]}
    wanted={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
    assert actual==wanted,"Published assets differ from tested package"
    assert api("git/ref/tags/"+TAG)["object"]["sha"]==sha
    assert api("releases/latest")["tag_name"]==TAG
    print("Published and verified:",release["html_url"])

if __name__=="__main__": main()
