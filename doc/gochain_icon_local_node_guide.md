# Gochain-icon local node guide

## References

- [Helper scripts to run gochain docker container as a local network](https://github.com/icon-project/gochain-local)
- [How to run goloop/testsuite with gochain docker image](https://gist.github.com/sink772/443b0abd0be1176b4ccb334205516450)
- [Testsuite README.md](https://github.com/icon-project/goloop/blob/master/testsuite/README.md)
- [Java SCORE Examples](https://github.com/icon-project/java-score-examples)
- [goloop_cli.md](https://github.com/icon-project/goloop/blob/master/doc/goloop_cli.md)  
- [Goloop JSON-RPC API v3](https://github.com/icon-project/goloop/blob/master/doc/jsonrpc_v3.md) 
- [JSON-RPC API v3 Extension for BTP](https://github.com/icon-project/goloop/blob/master/doc/btp_extension.md) 



## Goals

- You can build docker image for local.
- You can build javaee-api.jar for local.
- You can deploy sample java SCORE using goloop CLI binary.



## Requirements

- [Download and Install Docker](https://docs.docker.com/get-docker/)

- You need to install OpenJDK 11 version. Visit [OpenJDK.net](http://openjdk.java.net/) for prebuilt binaries. 
  Or you can install a proper OpenJDK package from your OS vendors.

  In macOS:

  ```
  $ brew tap AdoptOpenJDK/openjdk
  $ brew install --cask adoptopenjdk11
  ```

  In Linux (Ubuntu 18.04):

  ```
  $ sudo apt install openjdk-11-jdk
  ```



## Step 1. Source checkout

First of all, you need to checkout the `gochain-local` repository for executing local node.

```
$ git clone git@github.com:icon-project/gochain-local.git
$ GOCHAIN_LOCAL_ROOT=/path/to/gochain-local
```

Then, you need to checkout the `goloop` repository for building docker image and new javaee-api.jar.

```
$ git clone git@github.com:icon-project/goloop.git
$ GOLOOP_ROOT=/path/to/goloop
```

And last, you need to checkout the `java-score-examples` repository for sample java SCORE.

```
$ git clone git@github.com:icon-project/java-score-examples.git
$ JAVA_SCORE_EXAMPLES_ROOT=/path/to/java-score-examples
```



## Step 2. Build Docker image and goloop CLI for local

First of all, you need checkout git specific branch and run make file.

```
$ cd ${GOLOOP_ROOT}
$ git checkout master  # use the latest stable release
$ make gochain-icon-image
```

If the command runs successfully, it generates the docker image like the following.

```
$ docker images goloop/gochain-icon

REPOSITORY            TAG       IMAGE ID       CREATED          SIZE
goloop/gochain-icon   latest    73927da2b1a0   20 seconds ago   512MB
```

Then, you need build & copy built goloop binary file to`GOCHAIN_LOCAL_ROOT`.

```
$ make goloop
$ cp ./bin/goloop ${GOCHAIN_LOCAL_ROOT}/goloop
```



## Step 3. Build javaee-api.jar for local

If you want to use RLP method, you need to build `javaee-api.jar` for development.

```
$ cd ${GOLOOP_ROOT}/javaee
```

First of all, you can make `api-0.8.7-SNAPSHOT.jar` by using gradle script cmd.

```
$ ./gradlew api:build
```

If the command runs successfully, it generates the jar file on `./api/build/libs/`.

Then copy the jar file to `java-score-examples/hello-world`.

```
$ cp ./api/build/libs/api-0.8.7-SNAPSHOT.jar ${JAVA_SCORE_EXAMPLES_ROOT}/hello-world/api-0.8.7-SNAPSHOT.jar
```

If you want to get information about how to use RLP method, 
you can open local javadoc from `javaee/api/build/javadoc/index.html`.



## Step 4. Build sample java SCORE for local

```
$ cd ${JAVA_SCORE_EXAMPLES_ROOT}
```

First of all, edit `java-score-example/hello-world/build.gradle` as below.

```
...
dependencies {
    # use the local api jar for testing
    compile files('api-0.8.7-SNAPSHOT.jar')

    testImplementation 'org.junit.jupiter:junit-jupiter-api:5.6.0'
    testRuntimeOnly 'org.junit.jupiter:junit-jupiter-engine:5.6.0'
}
...
```

Prepare `hello-world/src/main/java/com/iconloop/score/example/HelloWorld.java` file as below.

```
/*
 * Copyright 2021 ICONLOOP Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.iconloop.score.example;

import score.Context;
import score.ObjectReader;
import score.ByteArrayObjectWriter;
import score.annotation.External;
import score.annotation.Payable;

public class HelloWorld {
    private final String name;

    public HelloWorld(String name) {
        this.name = name;
    }

    @External(readonly=true)
    public String name() {
        return name;
    }

    @External(readonly=true)
    public String getGreeting() {
        String msg = "Hello " + name + "!";
        Context.println(msg);
        return msg;
    }

    @Payable
    public void fallback() {
        // just receive incoming funds
    }

    @External
    public void testRlp() {
        var codec = "RLPn";
        var msg = "testRLP";

        ByteArrayObjectWriter w = Context.newByteArrayObjectWriter(codec);
        w.write(msg.getBytes());
        ObjectReader r = Context.newByteArrayObjectReader(codec, msg.getBytes());
    }
}
```

### Build the SCORE

```
$ ./gradlew build
```

The compiled jar bundle will be generated at `./hello-world/build/libs/hello-world-0.1.0.jar`.

### Optimize the jar

You need to optimize your jar bundle before you deploy it to local or ICON networks. 
This involves some pre-processing to ensure the actual deployment successful.

gradle-javaee-plugin is a Gradle plugin to automate the process of generating the optimized jar bundle. Run the optimizedJar task to generate the optimized jar bundle.

```
$ ./gradlew optimizedJar
```

The output jar will be located at `./hello-world/build/libs/hello-world-0.1.0-optimized.jar`.

### Copy example SCORE to GOCHAIN_LOCAL_ROOT

```
$ cp ./hello-world/build/libs/hello-world-0.1.0-optimized.jar ${GOCHAIN_LOCAL_ROOT}
```



## Step 5. Start gochain docker container

```
$ cd ${GOCHAIN_LOCAL_ROOT}
```

Prepare `run_gochain-icon.sh` file as below.

```
#!/bin/bash

usage() {
    echo "Usage: $0 [start|stop] (docker-tag)"
    exit 1
}

if [ $# -eq 1 ]; then
    CMD=$1
    TAG=latest
elif [ $# -eq 2 ]; then
    CMD=$1
    TAG=$2
else
    usage
fi

startDocker() {
    local dockerEnv=$1
    local port=$2
    echo ">>> START $dockerEnv $port $TAG"
    docker run -dit -v $PWD:/testsuite -p $port:$port \
        --env-file data/dockerenv/$dockerEnv \
        --name gochain-$dockerEnv \
        goloop/gochain-icon:$TAG
}

stopDocker() {
    echo ">>> STOP gochain-$1"
    docker stop gochain-$1
    docker rm gochain-$1
}

DOCKER_ENV=iconee
PORT=9082

case "$CMD" in
  start )
    startDocker $DOCKER_ENV $PORT
  ;;
  stop )
    stopDocker $DOCKER_ENV
  ;;
  * )
    echo "Error: unknown command: $CMD"
    usage
esac
```

 Start gochain-icon container.

```
$ ./run_gochain-icon.sh start
>>> START iconee 9082 latest
48e4c66fec68d01e767da91cbbb043c03f595b33cac69c8cdf94f39eaa03b34e

$ docker ps
CONTAINER ID   IMAGE                        COMMAND                  CREATED         STATUS         PORTS                                        NAMES
48e4c66fec68   goloop/gochain-icon:latest   "/entrypoint /bin/sh…"   9 seconds ago   Up 8 seconds   8080/tcp, 9080/tcp, 0.0.0.0:9082->9082/tcp   gochain-iconee
```

Note that log messages will be generated at `./chain/iconee.log`.

```
$ head ./chain/iconee.log
I|20210125-05:41:05.997850|b6b5|-|main|main.go:431   ____  ___   ____ _   _    _    ___ _   _
I|20210125-05:41:05.997953|b6b5|-|main|main.go:431  / ___|/ _ \ / ___| | | |  / \  |_ _| \ | |
I|20210125-05:41:05.997964|b6b5|-|main|main.go:431 | |  _| | | | |   | |_| | / _ \  | ||  \| |
I|20210125-05:41:05.997973|b6b5|-|main|main.go:431 | |_| | |_| | |___|  _  |/ ___ \ | || |\  |
I|20210125-05:41:05.997990|b6b5|-|main|main.go:431  \____|\___/ \____|_| |_/_/   \_\___|_| \_|
I|20210125-05:41:05.998006|b6b5|-|main|main.go:433 Version : v0.1.15-1039-g9f22c115
I|20210125-05:41:05.998057|b6b5|-|main|main.go:434 Build   : linux/amd64 tags()-2021-01-25-04:23:23
I|20210125-05:41:05.998094|b6b5|-|metric|metric.go:150 Initialize rootMetricCtx
T|20210125-05:41:05.998278|b6b5|-|TP|transport.go:383 registerPeerHandler &{0xc0001e6750 0xc0001e66f0 map[] {{0 0} 0 0 0 0} 0xc0001e67b0} true
T|20210125-05:41:05.998304|b6b5|-|TP|transport.go:383 registerPeerHandler &{0xc0001e66c0 :8080} true
```

### Stop the container (if test done)

```
$ ./run_gochain-icon.sh stop
>>> STOP gochain-iconee
gochain-iconee
gochain-iconee
```



## Step 6. Deploy the optimized jar (hello-world SCORE with RLP)

You can deploy the optimized jar by using goloop CLI binary.

```
$ cd ${GOCHAIN_LOCAL_ROOT}
$ ./goloop rpc sendtx deploy ./hello-world-0.1.0-optimized.jar \
    --uri http://localhost:9082/api/v3 \
    --key_store ./data/godWallet.json --key_password gochain \
    --nid 3 --step_limit 10000000000 \
    --content_type application/java \
    --param name=GoLoop
"0xfee1e31e3ecb88106e785a6cb8b0b957e42f5f908a7c2c66a0c19aebd659f7ef"
```

Check the deployed SCORE address first using the txresult command.

```
$ ./goloop rpc txresult 0xfee1e31e3ecb88106e785a6cb8b0b957e42f5f908a7c2c66a0c19aebd659f7ef \
    --uri http://localhost:9082/api/v3
{
  "to": "cx0000000000000000000000000000000000000000",
  "cumulativeStepUsed": "0x3d70a5c3",
  "stepUsed": "0x3d70a5c3",
  "stepPrice": "0x2e90edd00",
  "eventLogs": [],
  "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "status": "0x1",
  "scoreAddress": "cxd1f5d12e92459a4fcdf2678a14b572687471a70e",
  "blockHash": "0xe678a4a19d43c54c709c16b4d794e59a3214af64d9efd860eb8f97fc6bdece7d",
  "blockHeight": "0x1b6",
  "txIndex": "0x0",
  "txHash": "0xfee1e31e3ecb88106e785a6cb8b0b957e42f5f908a7c2c66a0c19aebd659f7ef"
}
```

Then you can query getGreeting method via the following call command.

```
$ ./goloop rpc call --to cxd1f5d12e92459a4fcdf2678a14b572687471a70e \
    --method getGreeting \
    --uri http://localhost:9082/api/v3
"Hello GoLoop!"
```

And you can invoke testRlp method via the following sendtx command.

```
$ ./goloop rpc sendtx call --to cxd1f5d12e92459a4fcdf2678a14b572687471a70e \
    --method testRlp \
    --uri http://localhost:9082/api/v3 \
    --key_store ./data/godWallet.json --key_password gochain \
    --nid 3 --step_limit 10000000000
"0x07868af25c42e0d201073eae9d490d895e0922a431918199aa3bd461d6d9e65f"
```

Check the called SCORE address using the txresult command.

```
$ ./goloop rpc txresult 0x07868af25c42e0d201073eae9d490d895e0922a431918199aa3bd461d6d9e65f \
    --uri http://localhost:9082/api/v3
{
  "to": "cxd1f5d12e92459a4fcdf2678a14b572687471a70e",
  "cumulativeStepUsed": "0x1fe85",
  "stepUsed": "0x1fe85",
  "stepPrice": "0x2e90edd00",
  "eventLogs": [],
  "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "status": "0x1",
  "blockHash": "0xae2be6e87b715f786c2ba599f5f570f9a4f018a20338cfb1d055ae3463d65571",
  "blockHeight": "0x2bc",
  "txIndex": "0x0",
  "txHash": "0x07868af25c42e0d201073eae9d490d895e0922a431918199aa3bd461d6d9e65f"
}
```
