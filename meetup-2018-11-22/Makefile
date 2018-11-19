build:
	docker build . -t jvassev/animal-logger:latest

push: build
	docker push jvassev/animal-logger:latest

run: build
	docker run -ti --rm jvassev/animal-logger:latest

