import os
from setuptools import setup, find_packages

with open('requirements.txt') as requirements:
    requires = list(requirements)

version = os.environ.get('VERSION')
if version is None:
    with open(os.path.join('.', 'VERSION')) as version_file:
        version = version_file.read().strip()

setup_options = {
    'name': 'pyexec',
    'version': version,
    'description': 'Python Execution Engine',
    'long_description_content_type': 'text/markdown',
    'long_description': open('README.md').read(),
    'url': 'https://github.com/icon-project/goloop/pyee/pyexec',
    'author': 'ICON Foundation',
    'author_email': 'foo@icon.foundation',
    'packages': find_packages(exclude=['tests*', 'docs']),
    'include_package_data': True,
    'license': "Apache License 2.0",
    'install_requires': requires,
    'python_requires': '>=3.6.5',
    'entry_points': {
        'console_scripts': [
            'pyexec=pyexec.__main__:main'
        ],
    },
    'classifiers': [
        'Development Status :: 5 - Production/Stable',
        'Intended Audience :: Developers',
        'Intended Audience :: System Administrators',
        'Natural Language :: English',
        'License :: OSI Approved :: Apache Software License',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7'
    ],
    'test_suite': 'tests'
}

setup(**setup_options)
