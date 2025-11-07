# Quick Start: Publishing Graft to PyPI

## 1. Install Tools
```bash
pip install build twine
```

## 2. Build Package
```bash
cd python
python -m build
```

## 3. Test Locally (Optional)
```bash
pip install dist/graft_orm-1.0.0-py3-none-any.whl
graft --help
```

## 4. Upload to PyPI

### First Time Setup
1. Create account: https://pypi.org/account/register/
2. Create API token: https://pypi.org/manage/account/token/
3. Save token securely

### Upload
```bash
twine upload dist/*
# Username: __token__
# Password: <your-api-token>
```

## 5. Verify
```bash
pip install graft-orm
graft --help
```

## Update Version
Edit these files before rebuilding:
- `setup.py` → VERSION = '1.0.1'
- `pyproject.toml` → version = "1.0.1"
- `graft_orm/__init__.py` → __version__ = "1.0.1"

Then:
```bash
rm -rf dist/ build/ *.egg-info
python -m build
twine upload dist/*
```

## GitHub Actions (Automated)
Add `PYPI_API_TOKEN` to GitHub Secrets, then releases will auto-publish to PyPI.
