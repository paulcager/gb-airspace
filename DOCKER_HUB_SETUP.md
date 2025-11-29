# Docker Hub Setup Guide

**Your workflow is already configured to publish to both GitHub Container Registry and Docker Hub!**

You just need to add two secrets to enable Docker Hub publishing.

## Step 1: Create a Docker Hub Account (if you don't have one)

1. Go to https://hub.docker.com/signup
2. Sign up (it's free!)
3. Verify your email

## Step 2: Create an Access Token

1. Log in to https://hub.docker.com
2. Click your username (top right) → **Account Settings**
3. Click **Security** → **New Access Token**
4. Token description: `GitHub Actions - gb-airspace`
5. Access permissions: **Read, Write, Delete**
6. Click **Generate**
7. **Copy the token** (you won't see it again!)

## Step 3: Add Secrets to GitHub

1. Go to https://github.com/paulcager/gb-airspace/settings/secrets/actions
2. Click **New repository secret**
3. Add first secret:
   - Name: `DOCKERHUB_USERNAME`
   - Value: Your Docker Hub username (e.g., `paulcager`)
   - Click **Add secret**
4. Click **New repository secret** again
5. Add second secret:
   - Name: `DOCKERHUB_TOKEN`
   - Value: The access token you copied in Step 2
   - Click **Add secret**

## That's It! 🎉

Your images will now be published to **both** registries:

### GitHub Container Registry
```bash
docker pull ghcr.io/paulcager/gb-airspace:latest
```

### Docker Hub
```bash
docker pull paulcager/gb-airspace:latest
```

## Verify It Worked

1. Go to https://github.com/paulcager/gb-airspace/actions
2. Click on the latest workflow run
3. You should see successful pushes to both registries

Then check:
- GitHub: https://github.com/paulcager/gb-airspace/pkgs/container/gb-airspace
- Docker Hub: https://hub.docker.com/r/paulcager/gb-airspace

## Making Images Public

### Docker Hub (automatic)
- Public by default for free accounts ✅

### GitHub Container Registry
1. Go to https://github.com/paulcager?tab=packages
2. Click `gb-airspace`
3. Package settings → Change visibility → Public

---

## Alternative: GitHub Container Registry Only

If you prefer to keep it simple and only use GitHub Container Registry, just use the original `docker-publish.yml` file (already committed). No extra setup needed!

Both approaches are **100% free** for public repositories.
