from setuptools import setup, find_packages
from setuptools.command.install import install
import subprocess
import sys
import os

VERSION = '2.0.8'

class PostInstallCommand(install):
    def run(self):
        install.run(self)
        install_script = os.path.join(self.install_lib, 'flashorm', 'install.py')
        if os.path.exists(install_script):
            subprocess.check_call([sys.executable, install_script])

def read_readme():
    readme_path = os.path.join(os.path.dirname(__file__), 'README.md')
    if os.path.exists(readme_path):
        with open(readme_path, 'r', encoding='utf-8') as f:
            return f.read()
    return 'A powerful, database-agnostic ORM with multi-database support and type-safe code generation'

setup(
    name='flashorm',
    version=VERSION,
    description='A powerful, database-agnostic ORM with multi-database support and type-safe code generation',
    long_description=read_readme(),
    long_description_content_type='text/markdown',
    author='Rana718',
    author_email='',
    url='https://github.com/Lumos-Labs-HQ/flash',
    packages=find_packages(),
    cmdclass={'install': PostInstallCommand},
    entry_points={
        'console_scripts': [
            'flash=flashorm.cli:main',
        ],
    },
    classifiers=[
        'Development Status :: 4 - Beta',
        'Intended Audience :: Developers',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Programming Language :: Python :: 3.11',
    ],
    python_requires='>=3.7',
    keywords='orm database migration postgresql mysql sqlite cli',
    project_urls={
        'Bug Reports': 'https://github.com/Lumos-Labs-HQ/flash/issues',
        'Source': 'https://github.com/Lumos-Labs-HQ/flash',
    },
)
