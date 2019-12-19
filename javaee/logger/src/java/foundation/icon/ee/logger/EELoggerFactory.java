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

import org.slf4j.ILoggerFactory;

import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

public class EELoggerFactory implements ILoggerFactory {
    private ConcurrentMap<String, org.slf4j.Logger> loggerMap;

    public EELoggerFactory() {
        loggerMap = new ConcurrentHashMap<>();
    }

    public org.slf4j.Logger getLogger(String name) {
        org.slf4j.Logger simpleLogger = loggerMap.get(name);
        if (simpleLogger != null) {
            return simpleLogger;
        } else {
            org.slf4j.Logger newInstance = new EELogger(name);
            org.slf4j.Logger oldInstance = loggerMap.putIfAbsent(name, newInstance);
            return oldInstance == null ? newInstance : oldInstance;
        }
    }

    /**
     * Clear the internal logger cache.
     *
     * This method is intended to be called by classes (in the same package) for
     * testing purposes. This method is internal. It can be modified, renamed or
     * removed at any time without notice.
     *
     * You are strongly discouraged from calling this method in production code.
     */
    void reset() {
        loggerMap.clear();
    }
}
