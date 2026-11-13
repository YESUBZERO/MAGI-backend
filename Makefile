SHELL := /usr/bin/env bash
KUBECONFIG ?= $(HOME)/.kube/config-k3s-tailscale

.PHONY: help ci bao-bootstrap build-push deploy-qa deploy-prod status argo-apply

help:
	@echo "MAGI DevOps / CI-CD Targets:"
	@echo "  make ci             - Ejecuta Go tests (-race), Next.js typecheck/tests y valida Kustomize"
	@echo "  make bao-bootstrap  - Configura OpenBao kv-v2, auth Kubernetes, políticas y secretos QA/Prod/CI"
	@echo "  make build-push     - Compila imágenes linux/arm64 y las publica en Harbor (ENV=qa|prod)"
	@echo "  make deploy-qa      - Ejecuta CI, build ARM64 -> Harbor y despliega en magi-qa"
	@echo "  make deploy-prod    - Ejecuta CI, build ARM64 -> Harbor y despliega en magi-prod"
	@echo "  make argo-apply     - Aplica las definiciones ArgoCD Application (magi-qa, magi-prod)"
	@echo "  make status         - Muestra el estado de ArgoCD, OpenBao, Runner, magi-qa y magi-prod"

ci:
	./scripts/pipeline-local.sh ci

bao-bootstrap:
	./scripts/openbao-bootstrap.sh

build-push:
	./scripts/pipeline-local.sh build-push $(or $(ENV),qa) $(TAG)

deploy-qa:
	./scripts/pipeline-local.sh deploy-qa $(or $(TAG),qa)

deploy-prod:
	./scripts/pipeline-local.sh deploy-prod $(or $(TAG),v0.2.0)

argo-apply:
	kubectl --kubeconfig $(KUBECONFIG) apply -f deploy/argocd/applications.yaml

status:
	./scripts/pipeline-local.sh status
