# kadrtt-test-plan
## KadRTT architecture
- Detailed description can be seen [here](https://hackmd.io/b-gKq_JmQLOSu1-v7IBRlw)
## How to install
- Install according to [here](https://docs.testground.ai/getting-started). Then go to ./testground and type `make install`, `go build`, and `sudo cp testground /usr/local/bin`.
- At $TESTGROUND_HOME/pkg/build/docker_go.go, add the following lines: 
~~~
# Copy only go.mod files and download deps, in order to leverage Docker caching.
# Add START
COPY /plan/go-libp2p-kad-dht ${PLAN_DIR}/go-libp2p-kad-dht
COPY /plan/go-libp2p-kbucket ${PLAN_DIR}/go-libp2p-kbucket
# ADD END
COPY /plan/go.mod ${PLAN_DIR}/go.mod
~~~
- Then type `make install` at $TESTGROUND_HOME, and `go build` .
- Copy the testground executable by `cp testground /usr/local/bin` for Ubuntu. 
- Pull the test plan by `git clone https://github.com/ncl-teu/kadrtt-test-plan` at $TESTGROUND_HOME. 
- Confirm that memory overcommitment is enabled. Add `sysctl vm.overcommit_memory=1` at /etc/sysctl.conf
- At $TESTGROUND_HOME/kadrtt-test-plan/, import the test-plan by `testground plan import --from dht/ --name dht`
- Then run the test-plan by `testground daemon` and in another terminal, type `testground run composition -f compositions/kadrtt.toml` at $TESTGROUND_HOME/kadrtt-test-plan/dht . 
- Increase ARP cache size for more than 40 or more container instances by varying kernel parameter as root:
- `# vi /etc/sysctl.conf` 
~~~
vm.overcommit_memory = 1
net.ipv4.neigh.default.gc_thresh1 = 2048
net.ipv4.neigh.default.gc_thresh2 = 4096
net.ipv4.neigh.default.gc_thresh3 = 8192
~~~
- Then see the current parameter by `syctl -p`. 
## Trouble shooting
- If goproxy is not working, type `docker run -d -p80:8081 goproxy/goproxy` or `docker system prune -a` and then `testground daemon`. 
- Or, see [here](https://docs.testground.ai/v/master/runner-library/local-docker/troubleshooting#troubleshooting)
