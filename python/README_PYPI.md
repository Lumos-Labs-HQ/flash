# Publishing Graft to PyPI

## Prerequisites

1. Install build tools:
```bash
pip install build twine
```

2. Create PyPI account at https://pypi.org/account/register/

3. Create API token at https://pypi.org/manage/account/token/

## Build Package

```bash
cd python
python -m build
```

This creates:
- `dist/graft-orm-1.0.0.tar.gz` (source distribution)
- `dist/graft_orm-1.0.0-py3-none-any.whl` (wheel)

## Test on TestPyPI (Recommended First)

```bash
# Upload to TestPyPI
twine upload --repository testpypi dist/*

# Test installation
pip install --index-url https://test.pypi.org/simple/ graft-orm
```

## Publish to PyPI

```bash
twine upload dist/*
```

Enter your PyPI username and API token when prompted.

## Verify Installation

```bash
pip install graft-orm
graft --help
```

## Update Version

1. Update version in:
   - `setup.py` (VERSION variable)
   - `pyproject.toml` ([project] version)
   - `graft_orm/__init__.py` (__version__)

2. Rebuild and republish:
```bash
rm -rf dist/ build/ *.egg-info
python -m build
twine upload dist/*
```

## Automation with GitHub Actions

Create `.github/workflows/pypi-release.yml`:

```yaml
name: Publish to PyPI

on:
  release:
    types: [published]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-python@v4
      with:
        python-version: '3.x'
    - name: Install dependencies
      run: |
        pip install build twine
    - name: Build package
      run: |
        cd python
        python -m build
    - name: Publish to PyPI
      env:
        TWINE_USERNAME: __token__
        TWINE_PASSWORD: ${{ secrets.PYPI_API_TOKEN }}
      run: |
        cd python
        twine upload dist/*
```

Add `PYPI_API_TOKEN` to GitHub repository secrets.
