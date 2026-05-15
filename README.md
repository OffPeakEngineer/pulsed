# psstd

## Install
```
# k8s
helm install psstd ./helm/psstd

# bare metal / LAN — just run the binary, mDNS does the rest
PSSTD_WEB=true ./psstd

# point a browser at any node
open http://<any-node-ip>:8080

```

## Discovery priority:
Environment	How peers are found
LAN / bare metal	mDNS _psstd._tcp — zero config
Kubernetes	PSSTD_SEEDS headless DNS (set by Helm)
Both	mDNS + seeds are merged, duplicates are fine
Single node	Runs solo, gets discovered when others start
