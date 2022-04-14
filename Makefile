SHELL := /bin/bash


help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install_generators: ## Intall generators
	@echo "Creating code-gen dir"
	GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	GO111MODULE=on go install k8s.io/code-generator/cmd/{defaulter-gen,client-gen,lister-gen,informer-gen,deepcopy-gen}@latest 

gen_crd_all: generate_crd_code generate_crd_manifests ## generate both code and manifests for crds

gen_crd_code: gen_crd_deep_copy ## generate deepcopy, listers, informers and cliensets for the types inside crd/pkg/apis/eventsrunner.io/v1alpha1

gen_crd_deep_copy:
	deepcopy-gen --input-dirs ./crd/pkg/apis/eventsrunner.io/v1alpha1 \
        -O zz_generated.deepcopy \
        --go-header-file ./hack/go-file-header.txt \
        -p github.com/luqmanMohammed/eventsrunner \
        -o ./

gen_crd_clientset_informer_lister:
	client-gen --clientset-name versioned --input-base '' \
        --input github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1 \
        --output-package github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset \
        --go-header-file ./hack/go-file-header.txt \
        -o ./
	lister-gen --input-dirs github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1 \
        --output-package github.com/luqmanMohammed/eventsrunner/crd/pkg/client/listers \
        --go-header-file ./hack/go-file-header.txt \
        -o ./
	informer-gen --input-dirs github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1 \
        --versioned-clientset-package github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned  \
        --listers-package github.com/luqmanMohammed/eventsrunner/crd/pkg/client/listers \
        --output-package github.com/luqmanMohammed/eventsrunner/crd/pkg/client/informers \
        --go-header-file ./hack/go-file-header.txt \
        -o ./
	rm -rf crd/pkg/client
	mv github.com/luqmanMohammed/eventsrunner/crd/pkg/client ./crd/pkg/client
	rm -rf github.com
gen_crd_manifests: ## generate manifests for the types inside crd/pkg/apis/eventsrunner.io/v1alpha1
	@echo "Generating manifests for crd/pkg/apis/eventsrunner.io/v1alpha1"
	controller-gen paths=./crd/pkg/apis/eventsrunner.io/v1alpha1 crd:crdVersions=v1 output:crd:artifacts:config=./crd/manifests

.DEFAULT_GOAL := all
.PHONY: run build clean deepclean test
