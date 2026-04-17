# Multi-stage build for optimized caching
FROM python:3.11-slim AS builder

# Set working directory
WORKDIR /app

# Install system dependencies (cached layer)
RUN apt-get update && apt-get install -y --no-install-recommends \
  git \
  curl \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean

# Copy requirements first (cached layer - only rebuilds when requirements change)
COPY requirements.txt /app/requirements.txt

# Install Python dependencies (cached layer)
RUN pip install --no-cache-dir --upgrade pip && \
  pip install --no-cache-dir -r requirements.txt

# Production stage
FROM python:3.11-slim AS production

# Set working directory
WORKDIR /docs

# Install curl for healthcheck
RUN apt-get update && apt-get install -y --no-install-recommends \
  curl \
  && rm -rf /var/lib/apt/lists/* \
  && apt-get clean

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin/mkdocs /usr/local/bin/mkdocs

# Create non-root user for security
RUN useradd -m -u 1000 mkdocs

# Set proper permissions
RUN chown -R mkdocs:mkdocs /docs

# Switch to non-root user
USER mkdocs

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8000/search/search_index.json || exit 1

# On-the-fly HTML via dev server (no static export); livereload on Markdown changes
CMD ["mkdocs", "serve", "--dev-addr=0.0.0.0:8000", "--livereload"]
