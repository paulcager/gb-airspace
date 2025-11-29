# GitHub Actions Workflows

## Docker Image Publishing

This repository automatically builds and publishes Docker images using GitHub Actions.

### How It Works

The `docker-publish.yml` workflow runs on:
- **Every push to `master`/`main`** - Builds and publishes image with `latest` tag
- **Every tag push (v*)** - Builds and publishes with semantic version tags
- **Pull requests** - Runs tests only (doesn't publish)

### Published Images

Images are published to **GitHub Container Registry (ghcr.io)**:

```bash
ghcr.io/paulcager/gb-airspace:latest
ghcr.io/paulcager/gb-airspace:master
ghcr.io/paulcager/gb-airspace:v1.2.3
```

### Usage

#### Pull the latest image:
```bash
docker pull ghcr.io/paulcager/gb-airspace:latest
```

#### Run the container:
```bash
docker run -p 9092:9092 ghcr.io/paulcager/gb-airspace:latest
```

### Creating a Release

To publish a versioned release:

```bash
# Tag your commit
git tag v1.0.0
git push origin v1.0.0
```

This will automatically build and publish:
- `ghcr.io/paulcager/gb-airspace:1.0.0`
- `ghcr.io/paulcager/gb-airspace:1.0`
- `ghcr.io/paulcager/gb-airspace:1`
- `ghcr.io/paulcager/gb-airspace:latest`

### Image Visibility

By default, GitHub Container Registry images are **private**. To make them public:

1. Go to https://github.com/paulcager/gb-airspace/pkgs/container/gb-airspace
2. Click "Package settings"
3. Scroll to "Danger Zone"
4. Click "Change visibility" → "Public"

### Multi-Platform Support

The workflow builds for both:
- `linux/amd64` (x86_64)
- `linux/arm64` (ARM, e.g., Raspberry Pi, M1/M2 Mac)

### Also Publishing to Docker Hub (Optional)

If you want to also publish to Docker Hub:

1. Create a Docker Hub access token at https://hub.docker.com/settings/security
2. Add it as a GitHub secret named `DOCKERHUB_TOKEN`
3. Add your Docker Hub username as `DOCKERHUB_USERNAME`
4. Uncomment the Docker Hub section in the workflow (see commented example below)

### Monitoring Builds

View build status and logs:
- https://github.com/paulcager/gb-airspace/actions
