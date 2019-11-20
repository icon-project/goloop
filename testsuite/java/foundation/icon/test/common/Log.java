package foundation.icon.test.common;

import java.util.EmptyStackException;
import java.util.Stack;

public class Log {
    private static final String[] PREFIX_LEVELS = {null, "[S]", "[W]", null, null};
    private static final String PREFIX_STEP_IN = "--> ";
    private static final String PREFIX_STEP_OUT = "<-- ";
    private static final String DEPTH_STRING = "   ";

    private static final int LEVEL_START = 0;
    public static final int LEVEL_NONE = LEVEL_START;
    public static final int LEVEL_SEVERE = LEVEL_NONE + 1;
    public static final int LEVEL_WARNING = LEVEL_SEVERE + 1;
    public static final int LEVEL_INFO = LEVEL_WARNING + 1;
    public static final int LEVEL_DEBUG = LEVEL_INFO + 1;
    private static final int LEVEL_END = LEVEL_DEBUG;

    private int level = LEVEL_INFO;
    private Stack<String> frames = new Stack<String>();

    public static Log getGlobal() {
        return new Log();
    }

    public void setLevel(int newLevel) {
        if (newLevel >= LEVEL_START && newLevel <= LEVEL_END) {
            level = newLevel;
        }
    }

    private boolean isLoggable(int level) {
        return this.level >= level && level > LEVEL_START;
    }

    public void info(String msg) {
        log(LEVEL_INFO, msg);
    }

    public void warning(String msg) {
        log(LEVEL_WARNING, msg);
    }

    public void severe(String msg) {
        log(LEVEL_SEVERE, msg);
    }

    public void infoEntering(String taskName, String msg) {
        if (taskName == null) {
            taskName = "";
        }
        if (msg == null) {
            msg = "";
        }
        StringBuilder buf = new StringBuilder(5 + taskName.length() + msg.length());
        buf.append(PREFIX_STEP_IN).append(taskName);
        if (msg.length() > 0) {
            buf.append(": ").append(msg);
        }
        log(LEVEL_INFO, buf.toString());
        frames.push(taskName);
    }

    public void infoEntering(String taskName) {
        infoEntering(taskName, null);
    }

    public void infoExiting(String msg) {
        if (msg == null) {
            msg = "";
        }
        try {
            String taskName = frames.pop();
            StringBuilder buf = new StringBuilder(5 + taskName.length() + msg.length());
            buf.append(PREFIX_STEP_OUT).append(taskName);
            if (msg.length() > 0) {
                buf.append(": ").append(msg);
            }
            log(LEVEL_INFO, buf.toString());
        } catch (EmptyStackException e) {
            log(LEVEL_WARNING, "(INVALID) Exiting without no entering" + msg);
        }
    }

    public void infoExiting() {
        infoExiting(null);
    }

    public void debug(String msg) {
        log(LEVEL_DEBUG, msg);
    }

    public void log(int level, String msg) {
        if (msg != null && isLoggable(level)) {
            if (PREFIX_LEVELS[level] != null || !frames.empty()) {
                StringBuilder buf = new StringBuilder(msg.length() + frames.size() * 3 + 3);
                for (int i = frames.size(); i > 0; i--) {
                    buf.append(DEPTH_STRING);
                }
                if (PREFIX_LEVELS[level] != null) {
                    buf.append(PREFIX_LEVELS[level]);
                }
                buf.append(msg);
                msg = buf.toString();
            }
            System.out.println(msg);
        }
    }
}
