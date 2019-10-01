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
import foundation.icon.ee.score.TransactionExecutor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ExecutorManager {
    private static final Logger logger = LoggerFactory.getLogger(ExecutorManager.class);
    private Map<String, TransactionExecutor> execMap;
    private ManagerProxy proxy;

    private String execSockAddr;

    public ExecutorManager(String sockAddr) throws IOException {
        Client client = Client.connect(sockAddr);
        proxy = new ManagerProxy(client);
        execSockAddr = sockAddr;
        execMap = new HashMap<>();

        proxy.setOnRunListener(this::runExecutor);
        proxy.setOnKillListener(this::killExecutor);
    }

    private void killExecutor(String uuid) throws IOException {
        logger.debug("[killExecutor]");
        TransactionExecutor executor = execMap.remove(uuid);
        if (executor != null) {
            logger.debug("disconnect executor uuid={}", uuid);
            executor.disconnect();
        }
    }

    private void runExecutor(String uuid) throws IOException {
        if (execMap.get(uuid) != null) {
            logger.info(uuid + " already exists");
            return;
        }
        TransactionExecutor exec = TransactionExecutor.newInstance(execSockAddr, uuid);
        execMap.put(uuid, exec);
        Thread th = new Thread(() -> {
            try {
                exec.connectAndRunLoop();
            } catch (IOException e) {
                logger.warn("executor terminated ", e);
            } finally {
                if (execMap.remove(uuid) != null) {
                    try {
                        proxy.end(uuid);
                    }
                    catch (IOException ex) {
                        logger.warn("Failed to send END message ", ex);
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