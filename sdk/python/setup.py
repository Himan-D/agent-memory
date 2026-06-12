from setuptools import setup, find_packages

setup(
    name="hystersis",
    version="2.0.0",
    description="Persistent memory infrastructure for AI agents",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    author="Hystersis Team",
    author_email="team@hystersis.com",
    url="https://hystersis.com",
    license="MIT",
    packages=find_packages(),
    install_requires=[
        "httpx>=0.25.0",
    ],
    extras_require={
        "integrations": [
            "requests>=2.28.0",
        ]
    },
    python_requires=">=3.9",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
    project_urls={
        "Homepage": "https://hystersis.com",
        "Documentation": "https://hystersis.com/docs",
        "Repository": "https://github.com/Himan-D/agent-memory",
        "Changelog": "https://hystersis.com/docs/changelog",
    },
)
