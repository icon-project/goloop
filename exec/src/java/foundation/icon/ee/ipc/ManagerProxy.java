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

    public ManagerProxy(Client client) {
        super(client, logger);
    }

    public void handleMessages() throws IOException {
        while (true) {
            Proxy.Message msg = getNextMessage();
            switch (msg.type) {
                case MsgType.RUN:
                    String uuid = msg.value.asStringValue().asString();
                    logger.debug("[RUN]");
                    handleRun(uuid);
                    break;
                case MsgType.KILL:
                    String uid = msg.value.asStringValue().asString();
                    logger.debug("[KILL]");
                    handleKill(uid);
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
        this.client.close();
    }

    public interface OnRunListener {
        void onRun(String uid) throws IOException;
    }

    public void setOnRunListener(OnRunListener listener) {
        mOnRunListener = listener;
    }

    public void handleRun(String uid) throws IOException {
        mOnRunListener.onRun(uid);
    }

    public interface OnKillListener {
        void onKill(String uid) throws IOException;
    }

    public void setOnKillListener(OnKillListener listener) {
        mOnKillListener = listener;
    }

    public void handleKill(String uid) throws IOException {
        mOnKillListener.onKill(uid);
    }

    public void end(String uuid) throws IOException {
        sendMessage(MsgType.END,  uuid);
    }
}
