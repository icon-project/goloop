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

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

public class ManagerProxy extends Proxy {
    private static final Logger logger = LoggerFactory.getLogger(ManagerProxy.class);
    private static final int VERSION = 1;

    private OnRunListener mOnRunListener;
    private OnKillListener mOnKillListener;

    static class MsgType {
        static final int VERSION = 100;
        static final int RUN = 101;
        static final int KILL = 102;
        static final int END = 103;
    }

    public ManagerProxy(Connection client) {
        super(client);
    }

    public void handleMessages() throws IOException {
        while (true) {
            Message msg = getNextMessage();
            switch (msg.type) {
                case MsgType.RUN:
                    String uuid = msg.value.asStringValue().asString();
                    logger.trace("[RUN] uuid={}", uuid);
                    handleRun(uuid);
                    break;
                case MsgType.KILL:
                    String uuid2 = msg.value.asStringValue().asString();
                    logger.trace("[KILL] uuid={}", uuid2);
                    handleKill(uuid2);
                    break;
                default:
                    break;
            }
        }
    }

    public void connect() throws IOException {
        sendMessage(MsgType.VERSION, VERSION, "java");
    }

    public void close() throws IOException {
        super.close();
    }

    public interface OnRunListener {
        void onRun(String uuid) throws IOException;
    }

    public void setOnRunListener(OnRunListener listener) {
        mOnRunListener = listener;
    }

    public void handleRun(String uuid) throws IOException {
        mOnRunListener.onRun(uuid);
    }

    public interface OnKillListener {
        void onKill(String uuid) throws IOException;
    }

    public void setOnKillListener(OnKillListener listener) {
        mOnKillListener = listener;
    }

    public void handleKill(String uuid) throws IOException {
        mOnKillListener.onKill(uuid);
    }

    public void end(String uuid) throws IOException {
        sendMessage(MsgType.END, uuid);
    }
}
