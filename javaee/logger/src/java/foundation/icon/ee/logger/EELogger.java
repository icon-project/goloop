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

package foundation.icon.ee.logger;

import foundation.icon.ee.ipc.EEProxy;
import org.slf4j.helpers.FormattingTuple;
import org.slf4j.helpers.MarkerIgnoringBase;
import org.slf4j.helpers.MessageFormatter;

import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.util.Map;

public class EELogger extends MarkerIgnoringBase {
    private String name;

    private static final int LOG_LEVEL_TRACE = 0;
    private static final int LOG_LEVEL_DEBUG = 1;
    private static final int LOG_LEVEL_INFO = 2;
    private static final int LOG_LEVEL_WARN = 3;
    private static final int LOG_LEVEL_ERROR = 4;

    private static String SYSTEM_PREFIX = "foundation.icon.ee.logger.";
    private static String LOG_LEVEL_KEY = SYSTEM_PREFIX + "defaultLogLevel";

    private static Map<String, Integer> LOG_MAP = Map.of(
        "trace", LOG_LEVEL_TRACE,
        "debug", LOG_LEVEL_DEBUG,
        "info", LOG_LEVEL_INFO,
        "warn", LOG_LEVEL_WARN,
        "error", LOG_LEVEL_ERROR,
        "fatal", LOG_LEVEL_ERROR,
        "panic", LOG_LEVEL_ERROR
    );

    private Map<Integer, Integer> PROXY_LOG_MAP = Map.of(
        LOG_LEVEL_TRACE, EEProxy.LOG_TRACE,
        LOG_LEVEL_DEBUG, EEProxy.LOG_DEBUG,
        LOG_LEVEL_INFO, EEProxy.LOG_INFO,
        LOG_LEVEL_WARN, EEProxy.LOG_WARN,
        LOG_LEVEL_ERROR, EEProxy.LOG_ERROR
    );

    private static int currentLogLevel = initLogLevel();
    private static int initLogLevel() {
        return LOG_MAP.getOrDefault(String.valueOf(System.getProperty(LOG_LEVEL_KEY)), LOG_LEVEL_INFO);
    }

    /**
     * For formatted messages, first substitute arguments and then log.
     *
     * @param level
     * @param format
     * @param arg1
     * @param arg2
     */
    private void formatAndLog(int level, String format, Object arg1, Object arg2) {
        if (!isLevelEnabled(level)) {
            return;
        }
        FormattingTuple tp = MessageFormatter.format(format, arg1, arg2);
        log(level, tp.getMessage(), tp.getThrowable());
    }

    /**
     * For formatted messages, first substitute arguments and then log.
     *
     * @param level
     * @param format
     * @param arguments
     *            a list of 3 ore more arguments
     */
    private void formatAndLog(int level, String format, Object... arguments) {
        if (!isLevelEnabled(level)) {
            return;
        }
        FormattingTuple tp = MessageFormatter.arrayFormat(format, arguments);
        log(level, tp.getMessage(), tp.getThrowable());
    }

    public boolean isLevelEnabled(int logLevel) {
        return (logLevel >= currentLogLevel);
    }

    public static int setLogLevel(int logLevel) {
        var res = currentLogLevel;
        currentLogLevel = logLevel;
        return res;
    }

    public EELogger(String name) {
        this.name = name;
    }

    private void log(int level, String message, Throwable t) {
        if (!isLevelEnabled(level)) {
            return;
        }
        StringBuilder strBuilder = new StringBuilder(String.valueOf(name));
        strBuilder.append(" ");
        if (t != null) {
            StringWriter sw = new StringWriter();
            PrintWriter pw = new PrintWriter(sw, true);
            pw.println(message);
            t.printStackTrace(pw);
            strBuilder.append(sw.getBuffer().toString());
        } else {
            strBuilder.append(message);
        }

        EEProxy proxy;
        if ((proxy = EEProxy.getProxy()) != null) {
            try {
                proxy.log(PROXY_LOG_MAP.getOrDefault(level, EEProxy.LOG_INFO), strBuilder.toString());
            } catch (IOException e) {
                e.printStackTrace();
            }
        } else {
            // use System.err for main thread
            System.err.printf("%s %s\n", renderLevel(level), strBuilder.toString());
            System.err.flush();
        }
    }

    protected String renderLevel(int level) {
        switch (level) {
            case LOG_LEVEL_TRACE:
                return "TRACE";
            case LOG_LEVEL_DEBUG:
                return ("DEBUG");
            case LOG_LEVEL_INFO:
                return "INFO";
            case LOG_LEVEL_WARN:
                return "WARN";
            case LOG_LEVEL_ERROR:
                return "ERROR";
        }
        throw new IllegalStateException("Unrecognized level [" + level + "]");
    }

    @Override
    public String getName() {
        return name;
    }

    @Override
    public boolean isTraceEnabled() {
        return isLevelEnabled(LOG_LEVEL_TRACE);
    }

    @Override
    public void trace(String msg) {
        log(LOG_LEVEL_TRACE, msg, null);
    }

    @Override
    public void trace(String format, Object arg) {
        formatAndLog(LOG_LEVEL_TRACE, format, arg, null);
    }

    @Override
    public void trace(String format, Object arg1, Object arg2) {
        formatAndLog(LOG_LEVEL_TRACE, format, arg1, arg2);
    }

    @Override
    public void trace(String format, Object... arguments) {
        formatAndLog(LOG_LEVEL_TRACE, format, arguments);
    }

    @Override
    public void trace(String msg, Throwable t) {
        log(LOG_LEVEL_TRACE, msg, t);
    }

    @Override
    public boolean isDebugEnabled() {
        return isLevelEnabled(LOG_LEVEL_DEBUG);
    }

    @Override
    public void debug(String msg) {
        log(LOG_LEVEL_DEBUG, msg, null);
    }

    @Override
    public void debug(String format, Object arg) {
        formatAndLog(LOG_LEVEL_DEBUG, format, arg, null);
    }

    @Override
    public void debug(String format, Object arg1, Object arg2) {
        formatAndLog(LOG_LEVEL_DEBUG, format, arg1, arg2);
    }

    @Override
    public void debug(String format, Object... arguments) {
        formatAndLog(LOG_LEVEL_DEBUG, format, arguments);
    }

    @Override
    public void debug(String msg, Throwable t) {
        log(LOG_LEVEL_DEBUG, msg, t);
    }

    @Override
    public boolean isInfoEnabled() {
        return isLevelEnabled(LOG_LEVEL_INFO);
    }

    @Override
    public void info(String msg) {
        log(LOG_LEVEL_INFO, msg, null);
    }

    @Override
    public void info(String format, Object arg) {
        formatAndLog(LOG_LEVEL_INFO, format, arg);
    }

    @Override
    public void info(String format, Object arg1, Object arg2) {
        formatAndLog(LOG_LEVEL_INFO, format, arg1, arg2);
    }

    @Override
    public void info(String format, Object... arguments) {
        formatAndLog(LOG_LEVEL_INFO, format, arguments);
    }

    @Override
    public void info(String msg, Throwable t) {
        log(LOG_LEVEL_INFO, msg, t);
    }

    @Override
    public boolean isWarnEnabled() {
        return isLevelEnabled(LOG_LEVEL_WARN);
    }

    @Override
    public void warn(String msg) {
        log(LOG_LEVEL_WARN, msg, null);
    }

    @Override
    public void warn(String format, Object arg) {
        formatAndLog(LOG_LEVEL_WARN, format, arg);
    }

    @Override
    public void warn(String format, Object... arguments) {
        formatAndLog(LOG_LEVEL_WARN, format, arguments);
    }

    @Override
    public void warn(String format, Object arg1, Object arg2) {
        formatAndLog(LOG_LEVEL_WARN, format, arg1, arg2);
    }

    @Override
    public void warn(String msg, Throwable t) {
        log(LOG_LEVEL_WARN, msg, t);
    }

    @Override
    public boolean isErrorEnabled() {
        return isLevelEnabled(LOG_LEVEL_ERROR);
    }

    @Override
    public void error(String msg) {
        log(LOG_LEVEL_ERROR, msg, null);
    }

    @Override
    public void error(String format, Object arg) {
        formatAndLog(LOG_LEVEL_ERROR, format, arg);
    }

    @Override
    public void error(String format, Object arg1, Object arg2) {
        formatAndLog(LOG_LEVEL_ERROR, format, arg1, arg2);
    }

    @Override
    public void error(String format, Object... arguments) {
        formatAndLog(LOG_LEVEL_ERROR, format, arguments);
    }

    @Override
    public void error(String msg, Throwable t) {
        log(LOG_LEVEL_ERROR, msg, t);
    }
}
