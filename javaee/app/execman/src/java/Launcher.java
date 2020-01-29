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

import foundation.icon.ee.ipc.Client;
import foundation.icon.ee.ipc.ExecutorManager;
import foundation.icon.ee.score.TransactionExecutor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

public class Launcher {
    public static void main(String[] args) throws IOException {
        Logger logger = LoggerFactory.getLogger(Launcher.class);
        if (args.length == 2) {
            TransactionExecutor executor = TransactionExecutor.newInstance(Client.connect(args[0]), args[1]);
            executor.connectAndRunLoop();
        } else if (args.length == 1) {
            ExecutorManager executorManager = new ExecutorManager(args[0]);
            executorManager.run();
        } else {
            logger.info("Usage: Launcher <socket addr> (<uuid>)");
        }
    }
}
