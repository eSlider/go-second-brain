# Virtual MkDocs pages from repo root (README, AGENTS). SDK docs live under docs/system/.
from pathlib import Path

import mkdocs_gen_files

ROOT = Path(__file__).resolve().parents[2]


def emit(src_rel: str, dest_rel: str) -> None:
    path = ROOT / src_rel
    if not path.is_file():
        return
    text = path.read_text(encoding="utf-8")
    with mkdocs_gen_files.open(dest_rel, "w") as f:
        f.write(text)


emit("README.md", "index.md")
emit("AGENTS.md", "AGENTS.md")
