# kadrtt-test-plan
## KadRTT architecture
- Detailed description can be seen [here](https://hackmd.io/b-gKq_JmQLOSu1-v7IBRlw)
## How to install
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
- At $TESTGROUND_HOME/kadrtt-test-plan/, import the test-plan by `testground plan import --from dht/ --name dht`
- Then run the test-plan by `testground daemon` and `testground run composition -f compositions/kadrtt.toml` at $TESTGROUND_HOME/kadrtt-test-plan/dht . 
