CLUSTER_NAME := kubelogin-system-test
CERT_DIR := ../cert

KUBECONFIG := kubeconfig.yaml
export KUBECONFIG

.PHONY: cluster
cluster:
	cp $(CERT_DIR)/ca.crt /tmp/kubelogin-system-test-dex-ca.crt
	kind create cluster --name $(CLUSTER_NAME) --config cluster.yaml
	echo '127.0.0.1 dex-server.localhost' | docker exec -i $(CLUSTER_NAME)-control-plane tee -a /etc/hosts
	# deploy Dex
	kubectl apply -f dex.yaml
	kubectl -n dex create secret tls dex.tls --cert=$(CERT_DIR)/server.crt --key=$(CERT_DIR)/server.key
	kubectl -n dex rollout status deployment server
	# wait for kube-apiserver oidc initialization
	# (oidc authenticator will retry oidc discovery every 10s)
	sleep 10
	kubectl -n kube-system logs --tail=10 kube-apiserver-$(CLUSTER_NAME)-control-plane
	# add the cluster role
	kubectl create clusterrole cluster-readonly --verb=get,watch,list --resource='*.*'
	kubectl create clusterrolebinding cluster-readonly --clusterrole=cluster-readonly --user=admin@example.com

.PHONY: terminate
terminate:
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: clean
clean:
	-rm $(KUBECONFIG)
