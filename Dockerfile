FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    make \
    zip \
    gnupg \
    tar \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (v20 LTS)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install LaTeX and Chromium dependencies for Puppeteer/Mermaid
RUN apt-get update && apt-get install -y \
    texlive-latex-base \
    texlive-xetex \
    texlive-fonts-recommended \
    texlive-extra-utils \
    texlive-latex-extra \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libcairo2 \
    libcups2 \
    libdbus-1-3 \
    libexpat1 \
    libfontconfig1 \
    libgbm1 \
    libglib2.0-0 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libpango-1.0-0 \
    libpangocairo-1.0-0 \
    libx11-xcb1 \
    libxcb-dri3-0 \
    libxcomposite1 \
    libxcursor1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxi6 \
    libxkbcommon0 \
    libxrandr2 \
    libxrender1 \
    libxss1 \
    libxtst6 \
    && rm -rf /var/lib/apt/lists/*

# Install Pandoc (v3.9.0.2)
RUN curl -L -o pandoc.deb https://github.com/jgm/pandoc/releases/download/3.9.0.2/pandoc-3.9.0.2-1-amd64.deb \
    && dpkg -i pandoc.deb \
    && rm pandoc.deb

# Install pandoc-crossref (v0.3.24a - matches pandoc v3.9.0.2)
RUN curl -L https://github.com/lierdakil/pandoc-crossref/releases/download/v0.3.24a/pandoc-crossref-Linux-X64.tar.xz | tar -xJ -C /usr/local/bin/

# Install pandoc-acro (acronym expansion filter, driven by the `acronyms`
# key in metadata.yaml)
RUN apt-get update && apt-get install -y python3 python3-pip \
    && pip install --no-cache-dir --break-system-packages pandoc-acro \
    && rm -rf /var/lib/apt/lists/*

# Install Vale (v3.15.1)
RUN curl -sfL https://github.com/errata-ai/vale/releases/download/v3.15.1/vale_3.15.1_Linux_64-bit.tar.gz | tar -xz -C /usr/local/bin vale

# Configure Puppeteer to use a global cache directory readable by all users
ENV PUPPETEER_CACHE_DIR=/usr/local/share/puppeteer

# Pre-install global npm packages to speed up formatting and mermaid rendering
RUN npm install -g prettier @mermaid-js/mermaid-cli@11.16.0

WORKDIR /workspace
