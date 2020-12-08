/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.ee.tooling;

import foundation.icon.ee.test.GoldenTest;
import foundation.icon.ee.util.MethodUnpacker;
import org.aion.avm.core.dappreading.LoadedJar;
import org.aion.avm.utilities.JarBuilder;
import org.junit.jupiter.api.Assertions;
import org.objectweb.asm.ClassReader;
import org.objectweb.asm.util.TraceClassVisitor;

import java.io.IOException;
import java.io.PrintWriter;
import java.util.Arrays;
import java.util.stream.Collectors;

public class OptimizerGoldenTest extends GoldenTest {
    public void test(Class<?>... args) {
        var jarBytes = makeRelJar(args);
        System.out.println();
        try {
            var apis = JarBuilder.getAPIsBytesFromJAR(jarBytes);
            var methods = MethodUnpacker.readFrom(apis);
            Arrays.asList(methods).forEach(System.out::println);
            System.out.println();

            var jar = LoadedJar.fromBytes(jarBytes);
            var keys = jar.classBytesByQualifiedNames
                    .keySet().stream().sorted().collect(Collectors.toList());
            for (var key : keys) {
                var bytes = jar.classBytesByQualifiedNames.get(key);
                var cr = new ClassReader(bytes);
                cr.accept(new TraceClassVisitor(new PrintWriter(System.out)), 0);
            }
        } catch (IOException e) {
            Assertions.fail(e);
        }
    }
}
