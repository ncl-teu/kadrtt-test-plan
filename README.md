# kadrtt-test-plan
## KadRTT architecture
- Detailed description can be seen [here](https://hackmd.io/b-gKq_JmQLOSu1-v7IBRlw)
## Installation of KadRTT
- Install according to [here](https://docs.testground.ai/getting-started).
- At $TESTGROUND_HOME/pkg/build/docker_go.go, add the following lines: 
~~~
# Copy only go.mod files and download deps, in order to leverage Docker caching.
# Add START
COPY /plan/go-libp2p-kad-dht ${PLAN_DIR}/go-libp2p-kad-dht
COPY /plan/go-libp2p-kbucket ${PLAN_DIR}/go-libp2p-kbucket
# ADD END
COPY /plan/go.mod ${PLAN_DIR}/go.mod
~~~
- Pull the test plan by `git clone https://github.com/ncl-teu/kadrtt-test-plan` at $TESTGROUND_HOME. 
- Then type `make install` at $TESTGROUND_HOME, and `go build` .
- Copy the testground executable by `cp testground /usr/local/bin` for Ubuntu. 
- At $TESTGROUND_HOME/kadrtt-test-plan/, import the test-plan by `testground plan import --from dht/ --name dht`
- Increase ARP cache size for up to 100 container instances(100 peers) by varying kernel parameter as root by `# vi /etc/sysctl.conf` to change the kernel parameters as follows: 
~~~
vm.overcommit_memory = 1
net.ipv4.neigh.default.gc_thresh1 = 4096
net.ipv4.neigh.default.gc_thresh2 = 8192
net.ipv4.neigh.default.gc_thresh3 = 16384
~~~
- Then see the current parameter by `syctl -p`. 
- Then refresh testground and pull some images as follows: 
~~~
docker system prune -a
docker pull iptestground/sidecar:edge
docker pull iptestground/sync-service:latest
~~~
- Then run the test-plan by `testground daemon` and in another terminal, type `testground run composition -f compositions/kadrttff100.toml` at $TESTGROUND_HOME/kadrtt-test-plan/dht for running 100 peers on single machine.
## Enable KadRTT
- At `dht/go-libpp-kbucket/table.go`, set as: 
~~~
	rt.isKadRTT = true
~~~
- If you disable KadRTT (i.e. kad-dht mode), set 
~~~
	rt.isKadRTT = false
~~~
- At .toml file in compositions/, add the following lines:
~~~
iskadrtt = "true"
kadrtt_interval = "180"
~~~
## Trouble shooting
- If goproxy is not working, type `docker run -d -p80:8081 goproxy/goproxy` or `docker system prune -a` and then `testground daemon`. 
- Or, see [here](https://docs.testground.ai/v/master/runner-library/local-docker/troubleshooting#troubleshooting)
- If peer containers are running due to unexceptional finish in tesground, kill them by `docker stop $(docker ps -q)`
