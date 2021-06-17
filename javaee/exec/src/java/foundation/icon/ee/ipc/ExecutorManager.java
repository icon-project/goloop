/*
 * Copyright 2019 ICON Foundation
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

package foundation.icon.ee.ipc;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import foundation.icon.ee.score.Loader;
import foundation.icon.ee.score.TransactionExecutor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExecutorManager {
    private static final Logger logger = LoggerFactory.getLogger(ExecutorManager.class);
    private final Map<String, TransactionExecutor> execMap;
    private final ManagerProxy proxy;
    private final String execSockAddr;
    private final Connector connector;
    private final Loader loader = new Loader();

    public ExecutorManager(String sockAddr, Connector c) throws IOException {
        Connection client = c.connect(sockAddr);
        proxy = new ManagerProxy(client);
        execSockAddr = sockAddr;
        execMap = new HashMap<>();
        connector = c;

        proxy.setOnRunListener(this::runExecutor);
        proxy.setOnKillListener(this::killExecutor);
    }

    public ExecutorManager(String sockAddr) throws IOException {
        this(sockAddr, Client.connector);
    }

    private void killExecutor(String uuid) throws IOException {
        TransactionExecutor executor = execMap.get(uuid);
        if (executor != null) {
            logger.trace("disconnect executor uuid={}", uuid);
            executor.disconnect();
        }
    }

    private void runExecutor(String uuid) {
        if (execMap.get(uuid) != null) {
            logger.info(uuid + " already exists");
            return;
        }
        Thread th = new Thread(() -> {
            try {
                TransactionExecutor exec = TransactionExecutor.newInstance(
                        connector.connect(execSockAddr),
                        uuid,
                        loader,
                        null,
                        null);
                execMap.put(uuid, exec);
                exec.connectAndRunLoop();
            } catch (Exception e) {
                System.err.println("Executor terminated: " + e);
                e.printStackTrace();
            } finally {
                if (execMap.remove(uuid) != null) {
                    try {
                        proxy.end(uuid);
                    }
                    catch (IOException ex) {
                        System.err.println("Failed to send END message: " + ex);
                    }
                }
            }
        });
        th.start();
    }

    public void run() throws IOException{
        proxy.connect();
        proxy.handleMessages();
        proxy.close();
    }
}