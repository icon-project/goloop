/*
 * Copyright 2023 ICON Foundation
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

package foundation.icon.ee.test;

import foundation.icon.ee.ipc.Connection;
import org.aion.avm.core.AvmConfiguration;

public class NoDebugTest extends SimpleTest {
    @Override
    public ServiceManager newServiceManager(Connection conn) {
        return new ServiceManager(conn, true);
    }

    @Override
    public AvmConfiguration newAvmConfiguration() {
        var conf = new AvmConfiguration();
        conf.testMode = true;
        conf.preserveDebuggability = false;
        return conf;
    }
}
