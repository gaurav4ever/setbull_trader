# python_strategies/setup.py
from setuptools import setup, find_packages

setup(
    name="mr_strategy",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        "pandas>=1.5.0",
        "numpy>=1.21.0",
        "requests>=2.27.1",
        "matplotlib>=3.5.1",
        "plotly>=5.10.0",
        "dash>=2.6.0",
        "python-dotenv>=0.20.0",
    ],
    author="Your Name",
    author_email="your.email@example.com",
    description="Morning Range Trading Strategy Implementation",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Financial and Insurance Industry",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
    ],
    python_requires=">=3.8",
)
