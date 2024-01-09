run:
 @echo "Starting Icecast Ripper Service"
 python src/main.py

test:
 @echo "Running tests"
 python -m unittest discover -s tests/

build:
 @echo "Building Docker image for Icecast Ripper Service"
 docker build -t icecast-ripper .

docker-run: build
 @echo "Running Icecast Ripper Service in a Docker container"
 docker run -p 8080:8080 --env-file .env.example icecast-ripper

clean:
 @echo "Cleaning up pycache and .pyc files"
 find . -type d -name pycache -exec rm -r {} +
 find . -type f -name '*.pyc' -delete

.PHONY: run test build docker-run clean
