# kadrtt-test-plan
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
