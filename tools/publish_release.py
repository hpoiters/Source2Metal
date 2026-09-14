"""Create v3.2.1 only; never replace any published tag or asset."""
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
REPO = "hpoiters/Source2Metal"
VERSION = "3.2.1"
PRESERVED = {
    "v3.0.9": ("24f23cf83a77654f256ea12e7cce13ef491096a4", {
        "SHA256_Source2Metal_v3.0.9.txt": "8f95209555803bd97541c5b2479bf6051cee6c2508d035c330b0eb728d1b6ca1",
        "Source2Metal_v3.0.9_RELEASE.zip": "7d6db6ac9d7d0d4410b83e8e1a5498827c04ffbd8819665adda3ac1853acf254"}),
    "v3.0.10": ("c27f18a5425658b31f06549b81f0633ae6464bf0", {
        "SHA256_Source2Metal_v3.0.10.txt": "f00f58b7128227996a0b10f53506d443adafa258f99fb7c385ee95515f6911e4",
        "Source2Metal_v3.0.10_RELEASE.zip": "dfd9443fcaf4e647ef56ee82c0c6f434a68f2359cb1f9c2060058c735b6db71b"}),
}


def api(path):
    return json.loads(subprocess.check_output(["gh", "api", f"repos/{REPO}/{path}"], text=True, encoding="utf-8"))


def preserved():
    snapshots = {}
    for tag, (sha, assets) in PRESERVED.items():
        assert api("git/ref/tags/" + tag)["object"]["sha"] == sha, f"Tag changed: {tag}"
        release = api("releases/tags/" + tag)
        actual = {a["name"]: a["digest"].removeprefix("sha256:") for a in release["assets"]}
        assert actual == assets, f"Assets changed: {tag}"
        snapshots[tag] = {k: release[k] for k in ("id", "tag_name", "name", "body", "draft", "prerelease", "published_at", "updated_at")}
    return snapshots


def main():
    assert os.environ["GITHUB_REF"] == "refs/heads/main"
    sha = os.environ["GITHUB_SHA"]
    before = preserved()
    assert api("git/ref/heads/main")["object"]["sha"] == sha, "Main moved during build"
    folder = ROOT / f"release_v{VERSION}"
    files = sorted(folder.iterdir())
    expected = {f"SHA256_Source2Metal_v{VERSION}.txt", f"Source2Metal_v{VERSION}_RELEASE.zip"}
    assert {p.name for p in files} == expected, "Unexpected release assets"
    tags = api("tags?per_page=100")
    assert all(t["name"] != "v" + VERSION for t in tags), "Version already tagged; refusing replacement"
    subprocess.run(["gh", "release", "create", "v" + VERSION, *map(str, files),
                    "--repo", REPO, "--target", sha, "--title", "Source2Metal v" + VERSION,
                    "--notes-file", str(ROOT / "source" / f"Source2Metal_v{VERSION}" / f"RELEASE_NOTES_v{VERSION}.txt"),
                    "--latest"], check=True)
    release = api("releases/tags/v" + VERSION)
    actual = {a["name"]: a["digest"].removeprefix("sha256:") for a in release["assets"]}
    expected_hashes = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
    assert actual == expected_hashes, "Published assets differ from tested package"
    assert api("git/ref/tags/v" + VERSION)["object"]["sha"] == sha
    assert api("releases/latest")["tag_name"] == "v" + VERSION
    assert preserved() == before, "An earlier release changed"
    print("Published and verified:", release["html_url"])


if __name__ == "__main__":
    main()
