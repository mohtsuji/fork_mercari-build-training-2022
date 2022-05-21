NAME=build2022/app
CONTAINER_NAME=build2022
VERSION=latest

build:
	docker image build -t $(NAME):$(VERSION) .

start:
	docker run -p 9000:9000 \
        --name $(CONTAINER_NAME) \
        $(NAME):$(VERSION)
       # -v xxx:xxx \

container=`docker ps -a -q`
image=`docker images | awk '/^<none>/ { print $$3 }'`

clean:
    @if [ "$(image)" != "" ] ; then \
        docker rmi $(image); \
    fi
    @if [ "$(container)" != "" ] ; then \
        docker rm $(container); \
    fi

stop:
	docker rm -f $(CONTAINER_NAME)

restart: stop start

attach:
	docker exec -it $(CONTAINER_NAME) sh

logs:
	docker logs $(CONTAINER_NAME)

.PHONY: build restart start clean stop attach logs