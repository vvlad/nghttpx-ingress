all: push

TAG = 0.1.1
PREFIX = docker.a1z.eu:5000/nghttpx-ingress

REPO_INFO=$(shell git config --get remote.origin.url)

ifndef VERSION
  VERSION := git-$(shell git rev-parse --short HEAD)
endif

controller: app/controller.go clean
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags \
		"-s -w -X main.version=${VERSION} -X main.gitRepo=${REPO_INFO}" \
		-o nghttpx-ingress

container: controller
	docker build --pull -t $(PREFIX):$(TAG) .

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f nghttpx-ingress
