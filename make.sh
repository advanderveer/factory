#!/bin/bash
set -e

DOCKERPWD=$(pwd)
if uname -a | grep Cygwin > /dev/null;
then
	DOCKERPWD=$(cygpath -aw $(pwd))
fi

STACK_ENV=${WHARF_ENV:=$(git config user.name)}

function print_help {
	printf "Available Commands:\n";
	awk -v sq="'" '/^function run_([a-zA-Z0-9-]*)\s*/ {print "-e " sq NR "p" sq " -e " sq NR-1 "p" sq }' make.sh \
		| while read line; do eval "sed -n $line make.sh"; done \
		| paste -d"|" - - \
		| sed -e 's/^/  /' -e 's/function run_//' -e 's/#//' -e 's/{/	/' \
		| awk -F '|' '{ print "  " $2 "\t" $1}' \
		| expand -t 30
}

function run_run { #generate API resources and interfaces
command -v docker >/dev/null 2>&1 || { echo "executable 'go' (the language sdk) must be installed: https://github.com/goadesign/goa" >&2; exit 1; }
	AWS_REGION=eu-west-1 AWS_PROFILE=phishermen go run main.go
}

function run_gen { #generate API resources and interfaces
command -v docker >/dev/null 2>&1 || { echo "executable 'goagen' (design-based api) must be installed: https://github.com/goadesign/goa" >&2; exit 1; }

	goagen app -d github.com/advanderveer/datajoin/backend/api/design --pkg app --out=api
}

function run_build { #compile the lambda function(s)
command -v docker >/dev/null 2>&1 || { echo "executable 'docker' (container runtime client) must be installed: https://www.docker.com/" >&2; exit 1; }
	P=github.com/advanderveer/datajoin

  echo "--> building (CLI)..."
  go build \
    -ldflags "-X main.version=$(cat VERSION) -X main.commit=$(git rev-parse --short HEAD )" \
    -o $GOPATH/bin/factory \
    main.go

	# echo "--> building..."
	# docker run --rm                                                             \
	#   -e HANDLER=handler                                                      	\
	#   -e PACKAGE=handler                                                      	\
	#   -v $DOCKERPWD:/go/src/$P/backend                															\
	#   -w /go/src/$P/backend                       															\
	#   eawsy/aws-lambda-go-shim:latest bash -c "go build -v -buildmode=plugin -ldflags='-w -s' -o handler.so; pack handler handler.so handler.zip"
  #
	# docker run --rm                                                              \
	#   -e HANDLER=handler                                                      	 \
	#   -e PACKAGE=handler                                                      	 \
	#   -v $DOCKERPWD:/go/src/$P/backend 																							 \
	#   -w /go/src/$P/backend/lambda/account_mailing                       				 \
	#   eawsy/aws-lambda-go-shim:latest bash -c "go build -v -buildmode=plugin -ldflags='-w -s' -o handler.so; pack handler handler.so handler.zip"
  #
	# docker run --rm                                                              \
	#   -e HANDLER=handler                                                      	 \
	#   -e PACKAGE=handler                                                      	 \
	#   -v $DOCKERPWD:/go/src/$P/backend 																					 \
	#   -w /go/src/$P/backend/lambda/phish_engine                       				 \
	#   eawsy/aws-lambda-go-shim:latest bash -c "go build -v -buildmode=plugin -ldflags='-w -s' -o handler.so; pack handler handler.so handler.zip"
  #
	# docker run --rm                                                              \
	#   -e HANDLER=handler                                                      	 \
	#   -e PACKAGE=handler                                                      	 \
	#   -v $DOCKERPWD:/go/src/$P/backend 																					 \
	#   -w /go/src/$P/backend/lambda/phish_tracking                       				 \
	#   eawsy/aws-lambda-go-shim:latest bash -c "go build -v -buildmode=plugin -ldflags='-w -s' -o handler.so; pack handler handler.so handler.zip"
}

function run_deploy { #deploy the stack through cloudformation
	command -v aws >/dev/null 2>&1 || { echo "executable 'aws' (AWS CLI) must be installed: https://aws.amazon.com/cli/" >&2; exit 1; }

	echo "--> validating..."
  aws cloudformation validate-template \
	 --profile=phishermen \
	 --region=eu-west-1 \
	 --template-body=file://formation.yaml

  #
	# echo "--> packaging..."
  # aws cloudformation package \
	# 	--profile=phishermen \
	# 	--region=eu-west-1 \
	# 	--template-file=formation.yaml \
	# 	--output-template-file=packaged.yaml \
	# 	--s3-bucket=phishermen-wharf \
	# 	--s3-prefix=$STACK_ENV
  #

	echo "--> deploying..."
  aws cloudformation deploy \
  	--profile=factory \
  	--region=eu-west-1 \
  	--template-file=formation.yaml \
  	--stack-name=factory \
  	--capabilities=CAPABILITY_IAM || true

}

function run_destroy { #remove the stack through cloudformation
	command -v aws >/dev/null 2>&1 || { echo "executable 'aws' (AWS CLI) must be installed: https://aws.amazon.com/cli/" >&2; exit 1; }

  aws cloudformation delete-stack \
	--profile=factory \
	--region=eu-west-1 \
	--stack-name=factory
}

case $1 in
	"run") run_run ;;
	"gen") run_gen ;;
  "build") run_build ;;
	"deploy") run_deploy ;;
  "destroy") run_destroy ;;
	*) print_help ;;
esac
