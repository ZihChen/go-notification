ECR_REPO=637423324672.dkr.ecr.ap-northeast-1.amazonaws.com/quantumcat
PLATFORM=linux/amd64

.PHONY: all login ssh-add build push rollout

all: build

login:
	aws ecr get-login-password --region ap-northeast-1 --profile jvd-demo \
	| docker login --username AWS --password-stdin ${ECR_REPO}

ssh-add:
	ssh-add ~/.ssh/id_rsa

build:
	docker buildx build \
		--platform ${PLATFORM} \
		--ssh default \
		-t ${ECR_REPO}/fatnotificationcat .

push:
	docker push ${ECR_REPO}/fatnotificationcat:latest

rollout:
	kubectl rollout restart deployment/fatnotificationcat-web -n bonuscat
	kubectl rollout restart deployment/fatnotificationcat-consumer -n bonuscat
	kubectl rollout restart deployment/fatnotificationcat-worker -n bonuscat
