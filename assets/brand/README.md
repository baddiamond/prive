# Brand assets

Logo, icons, banners and design tokens for Privé.

- **How to use them:** [`docs/brand/BRAND.md`](../../docs/brand/BRAND.md)
- **How they are made:** [`scripts/brand/generate.py`](../../scripts/brand/generate.py) — every file here except
  `tokens/tokens.json` and this README is generated. Edit the tokens or the generator, then run:

```bash
pip install -r scripts/brand/requirements.txt
python3 scripts/brand/generate.py
```

| Folder | Contents |
|---|---|
| `logo/` | Mark, wordmark and lockup — `on-dark`, `on-light`, `mono-white`, `mono-black` (SVG; PNG in `logo/png/`) |
| `icon/` | App icon SVG, PNG sizes 16–1024, `prive.icns` (macOS), `prive.ico` (Windows) |
| `tray/` | Menu bar template icon (macOS) and white tray icon (Windows/Linux) |
| `social/` | README banners (dark/light) and the GitHub social preview (1280×640) |
| `guide/` | Illustrations used by the brand guide |
| `tokens/` | `tokens.json` (source of truth), `tokens.css`, `tailwind.preset.js` |
