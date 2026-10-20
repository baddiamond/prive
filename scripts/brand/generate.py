#!/usr/bin/env python3
"""Generate every Privé brand asset from code.

Outputs (all committed to the repository):
  assets/brand/logo/      mark, wordmark and lockup SVGs + PNGs
  assets/brand/icon/      app icon SVG, PNG sizes, .icns and .ico
  assets/brand/tray/      menu bar / system tray icons
  assets/brand/social/    README banners and the GitHub social preview
  assets/brand/guide/     illustrations used by docs/brand/BRAND.md
  assets/brand/tokens/    tokens.css and tailwind.preset.js (from tokens.json)
  build/appicon.png       app icon used by Wails
  build/windows/icon.ico  Windows icon used by Wails

Usage:
  pip install -r scripts/brand/requirements.txt
  python3 scripts/brand/generate.py            # check contrast + generate everything
  python3 scripts/brand/generate.py --check    # only run the WCAG contrast checks

Text in SVGs is converted to outlines, so files render identically everywhere
without installing fonts. Inter (SIL Open Font License) is downloaded once into
.cache/brand/. PNGs are rendered with a local Chrome/Chromium in headless mode.
"""

from __future__ import annotations

import argparse
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile
import urllib.request
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
BRAND = ROOT / "assets" / "brand"
TOKENS_PATH = BRAND / "tokens" / "tokens.json"
CACHE = ROOT / ".cache" / "brand"
INTER_URL = "https://github.com/rsms/inter/releases/download/v4.1/Inter-4.1.zip"

# Mark geometry on a 64-unit grid. See docs/brand/BRAND.md#construction.
MARK_BOWL = {"cx": 34, "cy": 36, "r": 14}
MARK_STEM = ((20, 36), (20, 58))
MARK_ACCENT = ((39, 12), (49, 5))
MARK_STROKE = 7.0
MARK_ACCENT_STROKE = 6.5
# Tight bounding box of the mark including stroke caps: (min_x, min_y, max_x, max_y).
MARK_BOX = (16.5, 1.75, 52.25, 61.5)


# ---------------------------------------------------------------------------
# Tokens and contrast
# ---------------------------------------------------------------------------


def load_tokens() -> dict:
    """Read the design tokens that drive every generated file."""
    with TOKENS_PATH.open(encoding="utf-8") as f:
        return json.load(f)


def resolve(tokens: dict, dotted: str) -> str:
    """Resolve a dotted token path like 'theme.dark.text' to its value."""
    node = tokens
    for part in dotted.split("."):
        node = node[part]
    return node


def luminance(hex_color: str) -> float:
    """Relative luminance as defined by WCAG 2.x."""
    h = hex_color.lstrip("#")
    channels = [int(h[i : i + 2], 16) / 255 for i in (0, 2, 4)]
    linear = [c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4 for c in channels]
    return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]


def contrast(a: str, b: str) -> float:
    """WCAG contrast ratio between two hex colors."""
    hi, lo = sorted((luminance(a), luminance(b)), reverse=True)
    return (hi + 0.05) / (lo + 0.05)


def check_contrast(tokens: dict) -> bool:
    """Run every contrast rule. Returns True when all of them pass."""
    checks = [(c["fg"], c["bg"], c["min"]) for c in tokens["contrastChecks"]]
    # Account and status colors must work as text/fill in both themes.
    for group in ("account", "status"):
        for name, pair in tokens["color"][group].items():
            checks.append((f"color.{group}.{name}.dark", "theme.dark.bg", 4.5))
            checks.append((f"color.{group}.{name}.light", "theme.light.bg", 4.5))
    for name in tokens["color"]["account"]:
        checks.append(("theme.dark.onAccountColor", f"color.account.{name}.dark", 4.5))
        checks.append(("theme.light.onAccountColor", f"color.account.{name}.light", 4.5))

    ok = True
    for fg, bg, minimum in checks:
        ratio = contrast(resolve(tokens, fg), resolve(tokens, bg))
        passed = ratio >= minimum
        ok &= passed
        print(f"  {'pass' if passed else 'FAIL'}  {ratio:5.2f} >= {minimum:<3}  {fg} on {bg}")
    return ok


# ---------------------------------------------------------------------------
# Derived token files
# ---------------------------------------------------------------------------


def kebab(name: str) -> str:
    """Convert lowerCamelCase to kebab-case."""
    return "".join("-" + c.lower() if c.isupper() else c for c in name)


def write_tokens_css(tokens: dict) -> None:
    """Emit CSS custom properties. Dark is the default; light follows the OS or data-theme."""
    def block(theme: dict) -> str:
        return "\n".join(f"  --prive-{kebab(k)}: {v};" for k, v in theme.items())

    def account_block(mode: str) -> str:
        return "\n".join(
            f"  --prive-account-{name}: {pair[mode]};" for name, pair in tokens["color"]["account"].items()
        ) + "\n" + "\n".join(
            f"  --prive-status-{name}: {pair[mode]};" for name, pair in tokens["color"]["status"].items()
        )

    static = []
    for group in ("radius", "space", "fontSize"):
        for k, v in tokens[group].items():
            static.append(f"  --prive-{kebab(group)}-{kebab(k)}: {v};")
    static.append(f"  --prive-font-sans: {tokens['font']['sans']};")
    static.append(f"  --prive-font-mono: {tokens['font']['mono']};")
    for k, v in tokens["motion"].items():
        static.append(f"  --prive-motion-{kebab(k)}: {v};")

    dark = block(tokens["theme"]["dark"]) + "\n" + account_block("dark")
    light = block(tokens["theme"]["light"]) + "\n" + account_block("light")
    css = f"""/* Generated by scripts/brand/generate.py from tokens.json. Do not edit by hand. */

:root {{
{chr(10).join(static)}
}}

/* Dark is the default theme. */
:root,
:root[data-theme="dark"] {{
  color-scheme: dark;
{dark}
}}

/* Light theme: explicit choice, or the OS preference when no choice was made. */
:root[data-theme="light"] {{
  color-scheme: light;
{light}
}}

@media (prefers-color-scheme: light) {{
  :root:not([data-theme]) {{
    color-scheme: light;
{chr(10).join('  ' + line for line in light.splitlines())}
  }}
}}

@media (prefers-reduced-motion: reduce) {{
  :root {{
    --prive-motion-fast: 0ms;
    --prive-motion-base: 0ms;
  }}
}}
"""
    (BRAND / "tokens" / "tokens.css").write_text(css, encoding="utf-8")


def write_tailwind_preset(tokens: dict) -> None:
    """Emit a Tailwind preset that maps utilities to the CSS variables."""
    theme_keys = list(tokens["theme"]["dark"].keys())
    colors = ",\n".join(f'        "{kebab(k)}": "var(--prive-{kebab(k)})"' for k in theme_keys)
    accounts = ",\n".join(
        f'          {name}: "var(--prive-account-{name})"' for name in tokens["color"]["account"]
    )
    status = ",\n".join(f'          {name}: "var(--prive-status-{name})"' for name in tokens["color"]["status"])
    radius = ",\n".join(f'        {k}: "var(--prive-radius-{kebab(k)})"' for k in tokens["radius"])
    sizes = ",\n".join(f'        "{kebab(k)}": "var(--prive-font-size-{kebab(k)})"' for k in tokens["fontSize"])
    js = f"""// Generated by scripts/brand/generate.py from tokens.json. Do not edit by hand.
// Usage (frontend/tailwind.config.ts): presets: [require("../assets/brand/tokens/tailwind.preset.js")]
// and import "../assets/brand/tokens/tokens.css" once in the app entry point.

/** @type {{import('tailwindcss').Config}} */
module.exports = {{
  darkMode: ["selector", '[data-theme="dark"]'],
  theme: {{
    extend: {{
      colors: {{
{colors},
        account: {{
{accounts}
        }},
        status: {{
{status}
        }}
      }},
      borderRadius: {{
{radius}
      }},
      fontSize: {{
{sizes}
      }},
      fontFamily: {{
        sans: ["var(--prive-font-sans)"],
        mono: ["var(--prive-font-mono)"]
      }},
      transitionDuration: {{
        fast: "var(--prive-motion-fast)",
        base: "var(--prive-motion-base)"
      }},
      transitionTimingFunction: {{
        prive: "var(--prive-motion-easing)"
      }}
    }}
  }}
}};
"""
    (BRAND / "tokens" / "tailwind.preset.js").write_text(js, encoding="utf-8")


# ---------------------------------------------------------------------------
# Fonts and text outlines
# ---------------------------------------------------------------------------


def ensure_inter() -> Path:
    """Download Inter once and return the folder that holds its TTF files."""
    target = CACHE / "inter"
    ttf_dir = target / "extras" / "ttf"
    if (ttf_dir / "InterDisplay-Bold.ttf").exists():
        return ttf_dir
    print("Downloading Inter (SIL Open Font License)…")
    CACHE.mkdir(parents=True, exist_ok=True)
    with urllib.request.urlopen(INTER_URL) as resp:
        data = resp.read()
    with zipfile.ZipFile(io.BytesIO(data)) as z:
        z.extractall(target)
    return ttf_dir


class Typeface:
    """Shapes text with HarfBuzz (kerning included) and converts it to SVG paths."""

    def __init__(self, path: Path):
        import uharfbuzz as hb
        from fontTools.ttLib import TTFont

        self.font = TTFont(str(path))
        self.glyphs = self.font.getGlyphSet()
        self.upm = self.font["head"].unitsPerEm
        self.x_height = self.font["OS/2"].sxHeight
        self.cap_height = self.font["OS/2"].sCapHeight
        face = hb.Face(hb.Blob.from_file_path(str(path)))
        self.hb_font = hb.Font(face)

    def path(self, text: str, size: float, x: float, y: float, tracking: float = 0.0):
        """Return (svg_path_d, advance_width, (min_y, max_y)) for text on baseline y."""
        import uharfbuzz as hb
        from fontTools.pens.boundsPen import BoundsPen
        from fontTools.pens.svgPathPen import SVGPathPen
        from fontTools.pens.transformPen import TransformPen

        buf = hb.Buffer()
        buf.add_str(text)
        buf.guess_segment_properties()
        hb.shape(self.hb_font, buf, {"kern": True, "liga": True})
        scale = size / self.upm
        pen_x = 0.0
        parts, top, bottom = [], float("inf"), float("-inf")
        for info, pos in zip(buf.glyph_infos, buf.glyph_positions):
            name = self.font.getGlyphName(info.codepoint)
            ox = x + (pen_x + pos.x_offset) * scale
            oy = y - pos.y_offset * scale
            svg_pen = SVGPathPen(self.glyphs, ntos=lambda v: f"{v:.2f}".rstrip("0").rstrip("."))
            self.glyphs[name].draw(TransformPen(svg_pen, (scale, 0, 0, -scale, ox, oy)))
            parts.append(svg_pen.getCommands())
            bounds = BoundsPen(self.glyphs)
            self.glyphs[name].draw(bounds)
            if bounds.bounds:
                top = min(top, oy - bounds.bounds[3] * scale)
                bottom = max(bottom, oy - bounds.bounds[1] * scale)
            pen_x += pos.x_advance + tracking * self.upm
        width = (pen_x - tracking * self.upm) * scale
        return " ".join(p for p in parts if p), width, (top, bottom)


# ---------------------------------------------------------------------------
# SVG builders
# ---------------------------------------------------------------------------


def svg(width: float, height: float, body: str, title: str, view_box: str | None = None) -> str:
    """Wrap SVG content with an accessible title."""
    vb = view_box or f"0 0 {fmt(width)} {fmt(height)}"
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{fmt(width)}" height="{fmt(height)}" '
        f'viewBox="{vb}" role="img" aria-labelledby="title">\n'
        f"  <title id=\"title\">{title}</title>\n{body}\n</svg>\n"
    )


def fmt(v: float) -> str:
    """Format a number compactly for SVG attributes."""
    return f"{v:.2f}".rstrip("0").rstrip(".")


def mark_body(ink: str, accent: str, stroke: float = MARK_STROKE, accent_stroke: float = MARK_ACCENT_STROKE) -> str:
    """The mark's shapes in 64-unit grid coordinates."""
    b = MARK_BOWL
    (sx1, sy1), (sx2, sy2) = MARK_STEM
    (ax1, ay1), (ax2, ay2) = MARK_ACCENT
    return (
        f'    <circle cx="{b["cx"]}" cy="{b["cy"]}" r="{b["r"]}" fill="none" stroke="{ink}" stroke-width="{fmt(stroke)}"/>\n'
        f'    <path d="M{sx1} {sy1}V{sy2}" fill="none" stroke="{ink}" stroke-width="{fmt(stroke)}" stroke-linecap="round"/>\n'
        f'    <path d="M{ax1} {ay1}L{ax2} {ay2}" fill="none" stroke="{accent}" stroke-width="{fmt(accent_stroke)}" stroke-linecap="round"/>'
    )


def placed_mark(ink: str, accent: str, cx: float, cy: float, height: float, **kw) -> str:
    """The mark scaled to `height` and centered on (cx, cy)."""
    x0, y0, x1, y1 = MARK_BOX
    k = height / (y1 - y0)
    tx = cx - (x0 + x1) / 2 * k
    ty = cy - (y0 + y1) / 2 * k
    return f'  <g transform="translate({fmt(tx)} {fmt(ty)}) scale({fmt(k)})">\n{mark_body(ink, accent, **kw)}\n  </g>'


def mark_svg(ink: str, accent: str, title: str) -> str:
    """Standalone mark with a tight view box."""
    x0, y0, x1, y1 = MARK_BOX
    w, h = x1 - x0, y1 - y0
    return svg(w * 4, h * 4, f"  <g>\n{mark_body(ink, accent)}\n  </g>", title, f"{fmt(x0)} {fmt(y0)} {fmt(w)} {fmt(h)}")


def wordmark_parts(face: Typeface, ink: str, accent: str, size: float, x: float, baseline: float):
    """Paths for 'privé' with the 'é' in the accent color. Returns (svg, width, (top, bottom))."""
    tracking = -0.045
    d_main, w_main, ext1 = face.path("priv", size, x, baseline, tracking)
    # Advance to the é including the kerning pair "vé" by shaping the whole word.
    _, w_full, ext2 = face.path("privé", size, x, baseline, tracking)
    _, w_e, _ = face.path("é", size, 0, 0)
    e_x = x + w_full - w_e
    d_e, _, _ = face.path("é", size, e_x, baseline)
    body = f'  <path fill="{ink}" d="{d_main}"/>\n  <path fill="{accent}" d="{d_e}"/>'
    return body, w_full, (min(ext1[0], ext2[0]), max(ext1[1], ext2[1]))


def wordmark_svg(face: Typeface, ink: str, accent: str, title: str) -> str:
    """Standalone 'privé' wordmark."""
    size = 200
    body, width, (top, bottom) = wordmark_parts(face, ink, accent, size, 0, size)
    pad = size * 0.04
    return svg(width + 2 * pad, bottom - top + 2 * pad, body, title,
               f"{fmt(-pad)} {fmt(top - pad)} {fmt(width + 2 * pad)} {fmt(bottom - top + 2 * pad)}")


def lockup_parts(face: Typeface, ink: str, accent: str, size: float, x: float, baseline: float):
    """Mark + wordmark, with the mark's bowl centered on the wordmark's x-height."""
    x_height = face.x_height / face.upm * size
    bowl_outer = (MARK_BOWL["r"] + MARK_STROKE / 2) * 2  # in grid units
    k = x_height * 1.32 / bowl_outer
    mark_h = (MARK_BOX[3] - MARK_BOX[1]) * k
    mark_w = (MARK_BOX[2] - MARK_BOX[0]) * k
    bowl_center_y = baseline - x_height / 2
    # Offset between the mark's box center and its bowl center, scaled.
    box_center_y = (MARK_BOX[1] + MARK_BOX[3]) / 2
    cy = bowl_center_y - (MARK_BOWL["cy"] - box_center_y) * k
    mark = placed_mark(ink, accent, x + mark_w / 2, cy, mark_h)
    gap = size * 0.26
    word, word_w, (top, bottom) = wordmark_parts(face, ink, accent, size, x + mark_w + gap, baseline)
    top = min(top, cy - mark_h / 2)
    bottom = max(bottom, cy + mark_h / 2)
    return mark + "\n" + word, mark_w + gap + word_w, (top, bottom)


def lockup_svg(face: Typeface, ink: str, accent: str, title: str) -> str:
    """Standalone horizontal lockup."""
    size = 200
    body, width, (top, bottom) = lockup_parts(face, ink, accent, size, 0, size)
    pad = size * 0.06
    return svg(width + 2 * pad, bottom - top + 2 * pad, body, title,
               f"{fmt(-pad)} {fmt(top - pad)} {fmt(width + 2 * pad)} {fmt(bottom - top + 2 * pad)}")


def app_icon_svg(tile: str, border: str, ink: str, accent: str, title: str) -> str:
    """1024×1024 app icon on the macOS icon grid (824 px body, 100 px margin)."""
    body = (
        f'  <rect x="100" y="100" width="824" height="824" rx="185" fill="{tile}"/>\n'
        f'  <rect x="102" y="102" width="820" height="820" rx="183" fill="none" stroke="{border}" stroke-width="4"/>\n'
        + placed_mark(ink, accent, 512, 512, 824 * 0.62)
    )
    return svg(1024, 1024, body, title)


def tray_svg(color: str, title: str) -> str:
    """Single-color tray icon with heavier strokes for crisp rendering at 16 px."""
    return svg(64, 64, placed_mark(color, color, 32, 32, 60, stroke=8, accent_stroke=7.5), title)


def banner_svg(face_bold: Typeface, face_text: Typeface, theme: dict, title: str) -> str:
    """README banner: lockup, tagline and one-line description."""
    w, h = 1600, 520
    lock, lock_w, _ = lockup_parts(face_bold, theme["text"], theme["accent"], 150, 0, 0)
    tag = "Your conversations, at home."
    sub = "One private, local-first desktop inbox for all your messaging accounts."
    tag_d, tag_w, _ = face_bold.path(tag, 52, 0, 0, -0.02)
    sub_d, sub_w, _ = face_text.path(sub, 30, 0, 0)
    body = (
        f'  <rect width="{w}" height="{h}" rx="32" fill="{theme["bg"]}"/>\n'
        f'  <rect x="1" y="1" width="{w - 2}" height="{h - 2}" rx="31" fill="none" stroke="{theme["divider"]}" stroke-width="2"/>\n'
        f'  <g transform="translate({fmt((w - lock_w) / 2)} 220)">\n{lock}\n  </g>\n'
        f'  <path transform="translate({fmt((w - tag_w) / 2)} 350)" fill="{theme["text"]}" d="{tag_d}"/>\n'
        f'  <path transform="translate({fmt((w - sub_w) / 2)} 412)" fill="{theme["textSecondary"]}" d="{sub_d}"/>'
    )
    return svg(w, h, body, title)


def social_svg(face_bold: Typeface, face_text: Typeface, theme: dict) -> str:
    """GitHub social preview (1280×640)."""
    w, h = 1280, 640
    lock, lock_w, _ = lockup_parts(face_bold, theme["text"], theme["accent"], 140, 0, 0)
    tag_d, tag_w, _ = face_bold.path("Your conversations, at home.", 56, 0, 0, -0.02)
    sub_d, sub_w, _ = face_text.path("Local-first  ·  No account  ·  No telemetry  ·  Open source", 30, 0, 0)
    body = (
        f'  <rect width="{w}" height="{h}" fill="{theme["bg"]}"/>\n'
        f'  <g transform="translate({fmt((w - lock_w) / 2)} 270)">\n{lock}\n  </g>\n'
        f'  <path transform="translate({fmt((w - tag_w) / 2)} 410)" fill="{theme["text"]}" d="{tag_d}"/>\n'
        f'  <path transform="translate({fmt((w - sub_w) / 2)} 480)" fill="{theme["textSecondary"]}" d="{sub_d}"/>'
    )
    return svg(w, h, body, "Privé — Your conversations, at home.")


def palette_svg(face: Typeface, tokens: dict) -> str:
    """Color guide illustration for BRAND.md."""
    dark, light = tokens["theme"]["dark"], tokens["theme"]["light"]
    rows = [
        ("Dark theme", [(k, v) for k, v in dark.items() if k not in ("onAccountColor", "focus")], dark),
        ("Light theme", [(k, v) for k, v in light.items() if k not in ("onAccountColor", "focus")], light),
    ]
    w, sw, gap = 1600, 112, 10
    y = 40
    parts = [f'  <rect width="{w}" height="740" rx="24" fill="{dark["bg"]}"/>']
    for label, swatches, theme in rows:
        d, _, _ = face.path(label, 26, 40, y + 26)
        parts.append(f'  <path fill="{dark["text"]}" d="{d}"/>')
        y += 48
        for i, (name, value) in enumerate(swatches):
            x = 40 + i * (sw + gap)
            parts.append(f'  <rect x="{x}" y="{y}" width="{sw}" height="96" rx="12" fill="{value}" stroke="{dark["divider"]}" stroke-width="2"/>')
            n, _, _ = face.path(name, 15, x + 2, y + 122)
            hx, _, _ = face.path(value.upper(), 14, x + 2, y + 144)
            parts.append(f'  <path fill="{dark["text"]}" d="{n}"/>')
            parts.append(f'  <path fill="{dark["textSecondary"]}" d="{hx}"/>')
        y += 190
    d, _, _ = face.path("Account colors (dark / light)", 26, 40, y + 26)
    parts.append(f'  <path fill="{dark["text"]}" d="{d}"/>')
    y += 48
    for i, (name, pair) in enumerate(tokens["color"]["account"].items()):
        x = 40 + i * 250
        parts.append(f'  <rect x="{x}" y="{y}" width="110" height="80" rx="14" fill="{pair["dark"]}"/>')
        parts.append(f'  <rect x="{x + 118}" y="{y}" width="110" height="80" rx="14" fill="{pair["light"]}"/>')
        n, _, _ = face.path(name.capitalize(), 16, x + 2, y + 108)
        hx, _, _ = face.path(f'{pair["dark"]} / {pair["light"]}', 14, x + 2, y + 130)
        parts.append(f'  <path fill="{dark["text"]}" d="{n}"/>')
        parts.append(f'  <path fill="{dark["textSecondary"]}" d="{hx}"/>')
    return svg(w, 740, "\n".join(parts), "Privé color palette")


def construction_svg(face: Typeface, tokens: dict) -> str:
    """Mark construction on its 64-unit grid, with clear space."""
    dark = tokens["theme"]["dark"]
    s = 8  # pixels per grid unit
    pad = 64 * s / 4 + 64
    size = 64 * s + 2 * pad
    grid = []
    for i in range(0, 65, 4):
        p = pad + i * s
        grid.append(f'  <path d="M{p} {pad}V{pad + 64 * s}M{pad} {p}H{pad + 64 * s}" stroke="{dark["divider"]}" stroke-width="1"/>')
    clear = 64 * s / 4
    b = MARK_BOWL
    body = (
        f'  <rect width="{size}" height="{size}" rx="24" fill="{dark["bg"]}"/>\n'
        + "\n".join(grid)
        + f'\n  <rect x="{pad - clear}" y="{pad - clear}" width="{64 * s + 2 * clear}" height="{64 * s + 2 * clear}" fill="none" stroke="{dark["textFaint"]}" stroke-width="2" stroke-dasharray="10 8"/>'
        + f'\n  <g transform="translate({pad} {pad}) scale({s})">\n{mark_body(dark["text"], dark["accent"])}\n  </g>'
        + f'\n  <circle cx="{pad + b["cx"] * s}" cy="{pad + b["cy"] * s}" r="5" fill="{dark["accent"]}"/>'
    )
    label, _, _ = face.path("64-unit grid · bowl r14 · stroke 7 · clear space = 1/4 width", 18, 40, size - 24)
    body += f'\n  <path fill="{dark["textSecondary"]}" d="{label}"/>'
    return svg(size, size, body, "Privé mark construction")


# ---------------------------------------------------------------------------
# Rasterizing
# ---------------------------------------------------------------------------


def find_chrome() -> str:
    """Locate a Chrome/Chromium binary for headless rendering."""
    candidates = [
        os.environ.get("CHROME", ""),
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
        "/Applications/Chromium.app/Contents/MacOS/Chromium",
        "/opt/pw-browsers/chromium-1194/chrome-linux/chrome",
        shutil.which("chromium") or "",
        shutil.which("chromium-browser") or "",
        shutil.which("google-chrome") or "",
    ]
    for c in candidates:
        if c and Path(c).exists():
            return c
    sys.exit("Chrome or Chromium not found. Set CHROME=/path/to/chrome.")


def render_png(chrome: str, svg_path: Path, out: Path, width: int, height: int) -> None:
    """Render an SVG to a transparent PNG of the given size."""
    html = (
        "<!doctype html><html><head><style>html,body{margin:0;background:transparent}"
        "img{display:block}</style></head><body>"
        f'<img src="{svg_path.as_uri()}" width="{width}" height="{height}"></body></html>'
    )
    with tempfile.NamedTemporaryFile("w", suffix=".html", delete=False) as f:
        f.write(html)
        page = f.name
    try:
        out.parent.mkdir(parents=True, exist_ok=True)
        # Headless Chrome reserves part of the window for browser UI, so render a taller
        # window and crop to the exact size afterwards.
        subprocess.run(
            [chrome, "--headless=new", "--no-sandbox", "--disable-gpu", "--hide-scrollbars",
             "--allow-file-access-from-files", "--default-background-color=00000000",
             f"--window-size={max(width, 500)},{height + 400}", f"--screenshot={out}", Path(page).as_uri()],
            check=True, capture_output=True,
        )
        from PIL import Image

        with Image.open(out) as shot:
            shot.convert("RGBA").crop((0, 0, width, height)).save(out, optimize=True)
    finally:
        os.unlink(page)


def svg_size(path: Path) -> tuple[float, float]:
    """Read width/height attributes from a generated SVG."""
    import re

    head = path.read_text(encoding="utf-8")[:400]
    w = float(re.search(r'width="([\d.]+)"', head).group(1))
    h = float(re.search(r'height="([\d.]+)"', head).group(1))
    return w, h


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def write(path: Path, content: str) -> Path:
    """Write a text file, creating folders as needed."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return path


def generate(tokens: dict) -> None:
    """Build every asset."""
    from PIL import Image

    dark, light = tokens["theme"]["dark"], tokens["theme"]["light"]
    fonts = ensure_inter()
    bold = Typeface(fonts / "InterDisplay-Bold.ttf")
    text = Typeface(fonts / "Inter-Medium.ttf")

    write_tokens_css(tokens)
    write_tailwind_preset(tokens)

    logo = BRAND / "logo"
    variants = {
        "on-dark": (dark["text"], dark["accent"]),
        "on-light": (light["text"], light["accent"]),
        "mono-white": ("#FFFFFF", "#FFFFFF"),
        "mono-black": ("#000000", "#000000"),
    }
    for name, (ink, accent) in variants.items():
        write(logo / f"prive-mark-{name}.svg", mark_svg(ink, accent, "Privé logo mark"))
        write(logo / f"prive-wordmark-{name}.svg", wordmark_svg(bold, ink, accent, "Privé"))
        write(logo / f"prive-lockup-{name}.svg", lockup_svg(bold, ink, accent, "Privé"))

    icon = BRAND / "icon"
    write(icon / "prive-app-icon.svg", app_icon_svg(dark["raised"], tokens["color"]["neutral"]["700"], dark["text"], dark["accent"], "Privé app icon"))
    write(icon / "prive-app-icon-light.svg", app_icon_svg(light["surface"], light["divider"], light["text"], light["accent"], "Privé app icon, light"))

    tray = BRAND / "tray"
    write(tray / "prive-tray-template.svg", tray_svg("#000000", "Privé menu bar icon"))
    write(tray / "prive-tray-white.svg", tray_svg("#FFFFFF", "Privé tray icon, white"))

    social = BRAND / "social"
    write(social / "prive-banner-dark.svg", banner_svg(bold, text, dark, "Privé — Your conversations, at home."))
    write(social / "prive-banner-light.svg", banner_svg(bold, text, light, "Privé — Your conversations, at home."))
    write(social / "prive-social-preview.svg", social_svg(bold, text, dark))

    guide = BRAND / "guide"
    write(guide / "palette.svg", palette_svg(text, tokens))
    write(guide / "construction.svg", construction_svg(text, tokens))

    chrome = find_chrome()
    print("Rendering PNGs…")
    for name in variants:
        for kind, width in (("mark", 512), ("lockup", 1200)):
            src = logo / f"prive-{kind}-{name}.svg"
            w, h = svg_size(src)
            render_png(chrome, src, logo / "png" / f"prive-{kind}-{name}.png", width, round(width * h / w))

    icon_sizes = (16, 32, 64, 128, 256, 512, 1024)
    for size in icon_sizes:
        render_png(chrome, icon / "prive-app-icon.svg", icon / "png" / f"prive-app-icon-{size}.png", size, size)
    render_png(chrome, icon / "prive-app-icon-light.svg", icon / "png" / "prive-app-icon-light-1024.png", 1024, 1024)

    master = Image.open(icon / "png" / "prive-app-icon-1024.png").convert("RGBA")
    master.save(icon / "prive.icns")
    ico_sizes = [(s, s) for s in (16, 24, 32, 48, 64, 128, 256)]
    master.save(icon / "prive.ico", sizes=ico_sizes)

    for color in ("template", "white"):
        src = tray / f"prive-tray-{color}.svg"
        for size, suffix in ((16, ""), (32, "@2x"), (22, "-22"), (44, "-22@2x")):
            render_png(chrome, src, tray / "png" / f"prive-tray-{color}{suffix}.png", size, size)

    render_png(chrome, social / "prive-social-preview.svg", social / "prive-social-preview.png", 1280, 640)
    for theme in ("dark", "light"):
        render_png(chrome, social / f"prive-banner-{theme}.svg", social / f"prive-banner-{theme}.png", 1600, 520)

    build = ROOT / "build"
    build.mkdir(exist_ok=True)
    shutil.copyfile(icon / "png" / "prive-app-icon-1024.png", build / "appicon.png")
    (build / "windows").mkdir(exist_ok=True)
    shutil.copyfile(icon / "prive.ico", build / "windows" / "icon.ico")
    print("Done.")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--check", action="store_true", help="only run the contrast checks")
    args = parser.parse_args()

    tokens = load_tokens()
    print("Contrast checks (WCAG 2.2):")
    if not check_contrast(tokens):
        sys.exit("Contrast checks failed. Adjust tokens.json.")
    if not args.check:
        generate(tokens)


if __name__ == "__main__":
    main()
