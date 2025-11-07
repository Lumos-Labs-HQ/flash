from setuptools import setup, find_packages
from setuptools.command.install import install
import platform
import urllib.request
import os
import stat

VERSION = '1.0.0'
REPO = 'Lumos-Labs-HQ/graft'

class PostInstallCommand(install):
    def run(self):
        install.run(self)
        self.download_binary()
    
    def download_binary(self):
        system = platform.system().lower()
        machine = platform.machine().lower()
        
        platform_map = {'darwin': 'darwin', 'linux': 'linux', 'windows': 'win32'}
        arch_map = {'x86_64': 'x64', 'amd64': 'x64', 'arm64': 'arm64', 'aarch64': 'arm64'}
        
        mapped_platform = platform_map.get(system)
        mapped_arch = arch_map.get(machine)
        
        if not mapped_platform or not mapped_arch:
            raise Exception(f"Unsupported platform: {system}-{machine}")
        
        binary_name = 'graft.exe' if system == 'windows' else 'graft'
        download_name = f"graft-{mapped_platform}-{mapped_arch}{'.exe' if system == 'windows' else ''}"
        url = f"https://github.com/{REPO}/releases/download/v{VERSION}/{download_name}"
        
        bin_dir = os.path.join(os.path.dirname(__file__), 'graft_orm', 'bin')
        os.makedirs(bin_dir, exist_ok=True)
        binary_path = os.path.join(bin_dir, binary_name)
        
        print(f"📦 Installing Graft v{VERSION} for {system}-{machine}...")
        print(f"📥 Downloading from: {url}")
        
        urllib.request.urlretrieve(url, binary_path)
        os.chmod(binary_path, os.stat(binary_path).st_mode | stat.S_IEXEC)
        
        print("✅ Graft installed successfully!")

setup(
    name='graft-orm',
    version=VERSION,
    description='A powerful, database-agnostic ORM with multi-database support and type-safe code generation',
    long_description=open('../README.md').read(),
    long_description_content_type='text/markdown',
    author='Rana718',
    author_email='',
    url='https://github.com/Lumos-Labs-HQ/graft',
    packages=find_packages(),
    cmdclass={'install': PostInstallCommand},
    entry_points={
        'console_scripts': [
            'graft=graft_orm.cli:main',
        ],
    },
    classifiers=[
        'Development Status :: 4 - Beta',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: MIT License',
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
        'Bug Reports': 'https://github.com/Lumos-Labs-HQ/graft/issues',
        'Source': 'https://github.com/Lumos-Labs-HQ/graft',
    },
)
