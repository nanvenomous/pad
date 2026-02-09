The goal of this project is to make simple, self-hostable solution which syncs markdown notes between my desktop and android phone (progressive web app).

If there is a sync conflict I'd like to be able to jump to my computer and resolve it there (no need to resolve anything on the phone).
Ideally there are not conflicts too often as syncs should happen quickly.
I like the idea of git for conflict resolution but for the actual syncing I think git might be too slow. 
Primarily this thing should be easy to use and set up on android (syncthing is not, although it is a joy to set up on linux)

## Quick Start with Docker

Pull and run the latest production image:

```bash
docker pull ghcr.io/nanvenomous/pad:latest
docker run -p 4000:4000 -v ./notes:/data/notes ghcr.io/nanvenomous/pad:latest
```

Or use docker-compose for production:

```bash
docker compose -f docker-compose.prod.yml up -d
```

Or build locally:

```bash
task docker-build
task docker-run
```

## Development

For local development with hot-reloading:

```bash
docker compose up
```

### Testing

Run the integration test suite:

```bash
task test
```

The test suite covers:
- ✅ Core CRUD operations
- ✅ Conflict detection and resolution
- ✅ Folder management and move operations
- ✅ Real-time sync via WebSocket
- ✅ Filesystem watcher integration
- ✅ Concurrent update handling

See [TESTING.md](TESTING.md) for detailed testing documentation.

## Building for Production

Build the production Docker image:

```bash
task docker-build
```

Push to a registry (defaults to GitHub Container Registry):

```bash
task docker-push
```

To use a different registry:

```bash
REGISTRY=docker.io/yourusername task docker-push
```
