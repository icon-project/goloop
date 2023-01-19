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

import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import org.aion.avm.utilities.JarBuilder;

import java.util.Map;

public class Jars {
    public static byte[] make(String name, Class<?>[] all, boolean strip) {
        byte[] preopt = JarBuilder.buildJarForExplicitMainAndClasses(name, all);
        return new OptimizedJarBuilder(true, preopt, strip)
                .withUnreachableMethodRemover()
                .withRenamer().withLog(System.out).getOptimizedBytes();
    }

    public static byte[] make(String name, byte[] bc, boolean strip) {
        byte[] preopt = JarBuilder.buildJarForExplicitClassNamesAndBytecode(
                name, bc, Map.of());
        return new OptimizedJarBuilder(true, preopt, strip)
                .withUnreachableMethodRemover()
                .withRenamer().withLog(System.out).getOptimizedBytes();
    }

    public static byte[] make(Class<?> c) {
        return make(c.getName(), new Class<?>[]{c}, true);
    }
}
