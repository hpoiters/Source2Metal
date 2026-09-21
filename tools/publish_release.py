"""Create v3.2.1-r3 only; never replace any previously published tag or asset."""
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
REPO = "hpoiters/Source2Metal"
VERSION = "3.2.1"
TAG = "v3.2.1-r3"
PRESERVED = {
    "v3.2.1-r2": ("3954028a7d5396821c7d97facf6d3754d2c07dc0", {
        "SHA256_Source2Metal_v3.2.1.txt": "e56934c72f851e9b8976cb616fa374d691a470d9ea0397d975862842a8416411",
        "Source2Metal_v3.2.1_RELEASE.zip": "c6c3d06f4ac3237e2e7f5240fbac82adbba7142b987151b7048e15eb3461aefd"}),
    "v3.2.1": ("90b355757d849db71196e8df0ef4c61836d6d5a3", {
        "SHA256_Source2Metal_v3.2.1.txt": "932e75b1ee28dc556e2cbd5026a1ea07a9b8fec870163c2b8d7929868882e039",
        "Source2Metal_v3.2.1_RELEASE.zip": "6cec6df6e20a55e1680db38eb36533c7c1de3090a96dd2b874ae2fc88fd38b03"}),
    "v3.0.9": ("24f23cf83a77654f256ea12e7cce13ef491096a4", {
        "SHA256_Source2Metal_v3.0.9.txt": "8f95209555803bd97541c5b2479bf6051cee6c2508d035c330b0eb728d1b6ca1",
        "Source2Metal_v3.0.9_RELEASE.zip": "7d6db6ac9d7d0d4410b83e8e1a5498827c04ffbd8819665adda3ac1853acf254"}),
    "v3.0.10": ("c27f18a5425658b31f06549b81f0633ae6464bf0", {
        "SHA256_Source2Metal_v3.0.10.txt": "f00f58b7128227996a0b10f53506d443adafa258f99fb7c385ee95515f6911e4",
        "Source2Metal_v3.0.10_RELEASE.zip": "dfd9443fcaf4e647ef56ee82c0c6f434a68f2359cb1f9c2060058c735b6db71b"}),
    "v3.2.0": ("79115eced04ee8217d5e86a62fc0fefd48b51f13", {
        "SHA256_Source2Metal_v3.2.0.txt": "ae41987343d39c34d1ae60860c3c9a9e0ca9daf45c40cbc173be608571cb5f1b",
        "Source2Metal_v3.2.0_RELEASE.zip": "a84d637e606f08311b5b9ab5678ed0a9f9004e2fd539a115b4f91302e8614300"}),
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
    assert all(t["name"] != TAG for t in tags), "Version already tagged; refusing replacement"
    subprocess.run(["gh", "release", "create", TAG, *map(str, files),
                    "--repo", REPO, "--target", sha, "--title", "Source2Metal v" + VERSION + " (2CBH activity spinner)",
                    "--notes-file", str(ROOT / "source" / f"Source2Metal_v{VERSION}" / f"RELEASE_NOTES_v{VERSION}.txt"),
                    "--latest"], check=True)
    release = api("releases/tags/" + TAG)
    actual = {a["name"]: a["digest"].removeprefix("sha256:") for a in release["assets"]}
    expected_hashes = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
    assert actual == expected_hashes, "Published assets differ from tested package"
    assert api("git/ref/tags/" + TAG)["object"]["sha"] == sha
    assert api("releases/latest")["tag_name"] == TAG
    assert preserved() == before, "An earlier release changed"
    print("Published and verified:", release["html_url"])


if __name__ == "__main__":
    main()
