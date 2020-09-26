CERT_DIR := cert

.PHONY: login
login: setup
	$(MAKE) -C login

.PHONY: setup
setup: cluster setup-chrome

.PHONY: cluster
cluster: cert
	$(MAKE) -C cluster

.PHONY: setup-chrome
setup-chrome: cert
	# add the dex server certificate to the trust store
	if [ -d ~/.pki/nssdb ]; then certutil -A -d sql:$(HOME)/.pki/nssdb -n dex -i $(CERT_DIR)/ca.crt -t "TC,,"; fi

.PHONY: cert
cert:
	$(MAKE) -C cert

.PHONY: terminate
terminate:
	$(MAKE) -C cluster terminate

.PHONY: clean
clean:
	$(MAKE) -C cert clean
	$(MAKE) -C cluster clean
	$(MAKE) -C login clean
