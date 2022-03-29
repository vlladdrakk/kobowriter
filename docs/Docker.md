# Docker
In order to make building this project easier on different operating systems and distros this project includes a `Dockerfile` so we can build the project in the same environment anywhere. The docker image takes a long time to build, koxtoolchain takes a long time to build, but once built it doesn't need to be built again.
Note: The docker image is built with a specific version of go (currently 1.18). If you need it to be different, change the download link in the `Dockerfile` to what you need and rebuild the image.
## Setup
  1. Make sure your system is running docker (https://docs.docker.com/get-docker/)
  2. Build the docker image with `make docker_build`
  3. Build the project with `make docker`